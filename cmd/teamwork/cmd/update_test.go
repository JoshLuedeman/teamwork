package cmd

import (
	"strings"
	"testing"
)

func TestBuildSetupIssueBody_ContainsFileList(t *testing.T) {
	files := []string{"coder.agent.md", "tester.agent.md"}
	body := buildSetupIssueBody(files)

	for _, f := range files {
		want := "`.github/agents/" + f + "`"
		if !strings.Contains(body, want) {
			t.Errorf("body missing file reference %q", want)
		}
	}
}

func TestBuildSetupIssueBody_ContainsInstructions(t *testing.T) {
	body := buildSetupIssueBody([]string{"coder.agent.md"})

	for _, want := range []string{
		"/setup-teamwork",
		"<!-- CUSTOMIZE -->",
		"auto-detect",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing expected text %q", want)
		}
	}
}

func TestBuildSetupIssueBody_SingleFile(t *testing.T) {
	body := buildSetupIssueBody([]string{"architect.agent.md"})
	if !strings.Contains(body, "`.github/agents/architect.agent.md`") {
		t.Error("body missing architect.agent.md reference")
	}
}

func TestParseUpdateSource_Valid(t *testing.T) {
	owner, repo, err := parseUpdateSource("JoshLuedeman/teamwork")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "JoshLuedeman" || repo != "teamwork" {
		t.Errorf("got owner=%q repo=%q, want JoshLuedeman/teamwork", owner, repo)
	}
}

func TestParseUpdateSource_Invalid(t *testing.T) {
	cases := []string{"", "noslash", "/empty", "empty/", "/"}
	for _, c := range cases {
		_, _, err := parseUpdateSource(c)
		if err == nil {
			t.Errorf("parseUpdateSource(%q) should have returned error", c)
		}
	}
}
