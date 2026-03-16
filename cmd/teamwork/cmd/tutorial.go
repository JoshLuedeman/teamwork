package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var tutorialCmd = &cobra.Command{
	Use:   "tutorial",
	Short: "Guided walkthrough of a feature workflow",
	Long: `Walk through a mini feature workflow step-by-step to learn how
Teamwork orchestrates AI agents. This is a dry-run tutorial — no state
files are created and no LLM calls are made.

Use --non-interactive to print all steps at once (useful for CI or scripting).`,
	RunE: runTutorial,
}

func init() {
	tutorialCmd.Flags().Bool("non-interactive", false, "Print all steps without waiting for input")
	rootCmd.AddCommand(tutorialCmd)
}

// tutorialStep holds the pre-baked content for one step of the guided tutorial.
type tutorialStep struct {
	Title       string
	Role        string
	Description string
	Example     string
}

// tutorialSteps returns the ordered sequence of tutorial steps that mirror a
// feature workflow. The content is entirely pre-baked — no LLM calls needed.
func tutorialSteps() []tutorialStep {
	return []tutorialStep{
		{
			Title: "Step 1 — Define the Goal (Human)",
			Role:  "human",
			Description: `Every workflow starts with a human defining the goal.
You provide a short description of what you want to build.

  $ teamwork start feature "Add user avatar uploads"

This creates a new workflow, assigns it a unique ID, and sets the
first step in motion. The goal is recorded so every agent downstream
knows what success looks like.`,
			Example: `State transition:  CREATED → IN_PROGRESS
Current step:      1/9 — human: Create feature request
Output:            workflow ID, type, goal, status`,
		},
		{
			Title: "Step 2 — Plan the Work (Planner → Architect)",
			Role:  "planner",
			Description: `The Planner agent decomposes your goal into concrete tasks with
acceptance criteria. It produces a structured task list that later
agents consume.

The Architect then reviews feasibility, evaluates design tradeoffs,
and records any decisions as ADRs (Architecture Decision Records).

Each agent "hands off" to the next by writing a structured artifact
that the engine validates before advancing the workflow.`,
			Example: `Handoff artifact (Planner → Architect):
  {
    "tasks": [
      {"id": "t1", "title": "Add avatar column to users table"},
      {"id": "t2", "title": "Create upload endpoint"},
      {"id": "t3", "title": "Add avatar display component"}
    ],
    "acceptance_criteria": ["Avatars render on profile page", ...]
  }

State transition:  step 2 → step 3 (planner → architect)`,
		},
		{
			Title: "Step 3 — Implement and Test (Coder → Tester)",
			Role:  "coder",
			Description: `The Coder agent writes the implementation and opens a pull request.
It follows project conventions discovered from the codebase and
architecture decisions from the Architect.

The Tester then validates acceptance criteria with an adversarial
mindset — covering edge cases, error paths, and boundary conditions.

Quality gates fire between steps:
  ✓ handoff_complete — artifact exists and is well-formed
  ✓ tests_pass       — test suite passes before advancing
  ✓ lint_pass        — linter reports no new warnings`,
			Example: `Handoff artifact (Coder → Tester):
  Pull request #42 opened with:
    - 3 files changed, 147 insertions, 12 deletions
    - Unit tests included for new upload logic
    - Linked to originating task issue

State transition:  step 4 → step 5 (coder → tester)`,
		},
		{
			Title: "Step 4 — Review and Approve (Reviewer → Human)",
			Role:  "reviewer",
			Description: `After the Security Auditor scans for vulnerabilities, the Reviewer
checks the PR for correctness, quality, and standards compliance.

Then the workflow pauses for human approval. This is the checkpoint
where you decide whether to merge the PR. Teamwork will not advance
past this point without an explicit:

  $ teamwork approve <workflow-id>

This human-in-the-loop gate ensures nothing ships without your sign-off.`,
			Example: `  $ teamwork approve abc-123-def
  Workflow abc-123-def approved at step 8.
  Advancing to step 9 (documenter).

State transition:  step 7 → step 8 → step 9 (reviewer → human → documenter)`,
		},
		{
			Title: "Step 5 — Document and Complete (Documenter)",
			Role:  "documenter",
			Description: `The final agent updates documentation and the changelog to reflect
what was built. Once this step completes, the workflow transitions
to COMPLETED status.

You can review the full history at any time:

  $ teamwork status          # list active workflows
  $ teamwork history <id>    # see every handoff artifact
  $ teamwork logs <id>       # view detailed step logs

That's the complete lifecycle of a feature workflow! The same
state-machine model powers all 10 workflow types (bugfix, refactor,
hotfix, security-response, spike, release, rollback,
dependency-update, and documentation).`,
			Example: `State transition:  step 9 complete → COMPLETED
Final status:
  ID:     abc-123-def
  Type:   feature
  Status: completed
  Steps:  9/9`,
		},
	}
}

func runTutorial(cmd *cobra.Command, args []string) error {
	nonInteractive, err := cmd.Flags().GetBool("non-interactive")
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	steps := tutorialSteps()

	// Header
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "=== Teamwork Tutorial: Your First Feature Workflow ===")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "This tutorial walks through a feature workflow end-to-end.")
	fmt.Fprintln(out, "No files will be created and no LLM calls will be made.")
	fmt.Fprintln(out, "")

	if nonInteractive {
		return printAllSteps(out, steps)
	}
	return printInteractiveSteps(out, steps)
}

// printAllSteps outputs every step without pausing for input.
func printAllSteps(out io.Writer, steps []tutorialStep) error {
	for i, s := range steps {
		printStep(out, s)
		if i < len(steps)-1 {
			fmt.Fprintln(out, strings.Repeat("─", 60))
			fmt.Fprintln(out, "")
		}
	}
	printFooter(out)
	return nil
}

// printInteractiveSteps pauses between steps for user acknowledgment.
func printInteractiveSteps(out io.Writer, steps []tutorialStep) error {
	scanner := bufio.NewScanner(os.Stdin)
	for i, s := range steps {
		printStep(out, s)
		if i < len(steps)-1 {
			fmt.Fprintln(out, "")
			fmt.Fprint(out, "Press Enter to continue...")
			scanner.Scan()
			fmt.Fprintln(out, strings.Repeat("─", 60))
			fmt.Fprintln(out, "")
		}
	}
	printFooter(out)
	return nil
}

// printStep renders a single tutorial step to the writer.
func printStep(out io.Writer, s tutorialStep) {
	fmt.Fprintf(out, ">> %s\n", s.Title)
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, s.Description)
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, s.Example)
	fmt.Fprintln(out, "")
}

// printFooter renders the closing message of the tutorial.
func printFooter(out io.Writer) {
	fmt.Fprintln(out, "=== Tutorial Complete ===")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Ready to start for real? Try:")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "  $ teamwork start feature \"Add user avatar uploads\"")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Run 'teamwork help' to explore all available commands.")
}
