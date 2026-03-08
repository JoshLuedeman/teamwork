// Package github integrates with GitHub for issue and PR linking
// by wrapping the gh CLI. This avoids auth complexity — gh handles
// authentication natively.
package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/JoshLuedeman/teamwork/internal/config"
)

// Client communicates with GitHub via the gh CLI.
type Client struct {
	Owner string
	Repo  string
}

// Issue represents a GitHub issue.
type Issue struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	State  string   `json:"state"`
	Labels []string `json:"labels"`
	Body   string   `json:"body"`
}

// PullRequest represents a GitHub pull request.
type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Branch string `json:"headRefName"`
	URL    string `json:"url"`
}

// issueJSON mirrors the JSON output from gh for issues, where labels are
// objects with a "name" field rather than plain strings.
type issueJSON struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Body string `json:"body"`
}

func (ij *issueJSON) toIssue() Issue {
	labels := make([]string, len(ij.Labels))
	for i, l := range ij.Labels {
		labels[i] = l.Name
	}
	return Issue{
		Number: ij.Number,
		Title:  ij.Title,
		State:  ij.State,
		Labels: labels,
		Body:   ij.Body,
	}
}

// NewClient creates a Client for the given owner and repo.
func NewClient(owner, repo string) *Client {
	return &Client{Owner: owner, Repo: repo}
}

// NewClientFromConfig parses "owner/repo" from the project config.
func NewClientFromConfig(cfg *config.Config) (*Client, error) {
	parts := strings.SplitN(cfg.Project.Repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("github: invalid repo format %q, expected \"owner/repo\"", cfg.Project.Repo)
	}
	return NewClient(parts[0], parts[1]), nil
}

// Available reports whether the gh CLI is installed and accessible.
func (c *Client) Available() bool {
	err := exec.Command("gh", "--version").Run()
	return err == nil
}

// GetIssue fetches a single issue by number.
func (c *Client) GetIssue(number int) (*Issue, error) {
	out, err := c.runGH("issue", "view", strconv.Itoa(number),
		"--json", "number,title,state,labels,body")
	if err != nil {
		return nil, fmt.Errorf("github: get issue #%d: %w", number, err)
	}

	var raw issueJSON
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("github: parse issue #%d: %w", number, err)
	}
	issue := raw.toIssue()
	return &issue, nil
}

// ListIssues lists issues filtered by state and optional labels.
func (c *Client) ListIssues(state string, labels []string) ([]Issue, error) {
	args := []string{"issue", "list", "--json", "number,title,state,labels", "--state", state}
	for _, l := range labels {
		args = append(args, "--label", l)
	}

	out, err := c.runGH(args...)
	if err != nil {
		return nil, fmt.Errorf("github: list issues: %w", err)
	}

	var raw []issueJSON
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("github: parse issue list: %w", err)
	}

	issues := make([]Issue, len(raw))
	for i, r := range raw {
		issues[i] = r.toIssue()
	}
	return issues, nil
}

// GetPR fetches a single pull request by number.
func (c *Client) GetPR(number int) (*PullRequest, error) {
	out, err := c.runGH("pr", "view", strconv.Itoa(number),
		"--json", "number,title,state,headRefName,url")
	if err != nil {
		return nil, fmt.Errorf("github: get PR #%d: %w", number, err)
	}

	var pr PullRequest
	if err := json.Unmarshal(out, &pr); err != nil {
		return nil, fmt.Errorf("github: parse PR #%d: %w", number, err)
	}
	return &pr, nil
}

// AddLabel adds a label to an issue or PR.
func (c *Client) AddLabel(issueNumber int, label string) error {
	_, err := c.runGH("issue", "edit", strconv.Itoa(issueNumber), "--add-label", label)
	if err != nil {
		return fmt.Errorf("github: add label %q to #%d: %w", label, issueNumber, err)
	}
	return nil
}

// RemoveLabel removes a label from an issue or PR.
func (c *Client) RemoveLabel(issueNumber int, label string) error {
	_, err := c.runGH("issue", "edit", strconv.Itoa(issueNumber), "--remove-label", label)
	if err != nil {
		return fmt.Errorf("github: remove label %q from #%d: %w", label, issueNumber, err)
	}
	return nil
}

// SetWorkflowLabel sets a workflow status label (e.g. "workflow:feature:active")
// and removes any existing status labels for the same workflow type.
func (c *Client) SetWorkflowLabel(issueNumber int, workflowType, status string) error {
	// Fetch current labels to find old ones to remove.
	issue, err := c.GetIssue(issueNumber)
	if err != nil {
		return fmt.Errorf("github: set workflow label: %w", err)
	}

	prefix := "workflow:" + workflowType + ":"
	for _, l := range issue.Labels {
		if strings.HasPrefix(l, prefix) {
			if err := c.RemoveLabel(issueNumber, l); err != nil {
				return err
			}
		}
	}

	newLabel := prefix + status
	return c.AddLabel(issueNumber, newLabel)
}

// CreateIssue creates a new GitHub issue with the given title, body, labels,
// and assignees. It returns the issue number on success.
func (c *Client) CreateIssue(title, body string, labels, assignees []string) (int, error) {
	args := []string{"issue", "create", "--title", title, "--body", body}
	for _, l := range labels {
		args = append(args, "--label", l)
	}
	for _, a := range assignees {
		args = append(args, "--assignee", a)
	}

	out, err := c.runGH(args...)
	if err != nil {
		return 0, fmt.Errorf("github: create issue: %w", err)
	}

	// gh issue create outputs the issue URL, e.g.
	// https://github.com/owner/repo/issues/42
	url := strings.TrimSpace(string(out))
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return 0, fmt.Errorf("github: unexpected issue create output: %q", url)
	}
	num, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, fmt.Errorf("github: parse issue number from %q: %w", url, err)
	}
	return num, nil
}

// runGH executes the gh CLI with the given arguments, scoped to the
// configured repository via -R owner/repo. It returns stdout on success.
func (c *Client) runGH(args ...string) ([]byte, error) {
	fullArgs := append([]string{"-R", c.Owner + "/" + c.Repo}, args...)
	cmd := exec.Command("gh", fullArgs...)

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh %s: %s", strings.Join(args, " "), string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}
