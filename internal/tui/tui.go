// Package tui provides an interactive terminal dashboard for viewing and
// navigating workflow state using bubbletea.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/JoshLuedeman/teamwork/internal/state"
	"github.com/JoshLuedeman/teamwork/internal/workflow"
)

// Styles used throughout the dashboard.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Padding(0, 1)

	statusActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42"))

	statusBlockedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("203"))

	statusCompletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	detailBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(1, 2)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)
)

// Model holds the TUI state.
type Model struct {
	engine    *workflow.Engine
	workflows []*state.WorkflowState
	cursor    int
	view      string // "list" or "detail"
	selected  *state.WorkflowState
	width     int
	height    int
	err       error
}

// refreshMsg signals the model to reload workflow data.
type refreshMsg struct{}

func initialModel(engine *workflow.Engine) Model {
	return Model{
		engine: engine,
		view:   "list",
	}
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg { return refreshMsg{} }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case refreshMsg:
		workflows, err := m.engine.Status()
		if err != nil {
			m.err = err
			return m, nil
		}
		m.workflows = workflows
		m.err = nil
		if m.cursor >= len(m.workflows) {
			m.cursor = max(0, len(m.workflows)-1)
		}
		// Refresh selected if in detail view.
		if m.view == "detail" && m.selected != nil {
			m.selected = m.findWorkflow(m.selected.ID)
		}
		return m, nil

	case tea.KeyMsg:
		switch {
		case msg.String() == "ctrl+c" || msg.String() == "q":
			return m, tea.Quit

		case msg.String() == "r":
			return m, func() tea.Msg { return refreshMsg{} }

		case msg.String() == "esc":
			if m.view == "detail" {
				m.view = "list"
				m.selected = nil
			}
			return m, nil

		case msg.String() == "enter":
			if m.view == "list" && len(m.workflows) > 0 {
				m.selected = m.workflows[m.cursor]
				m.view = "detail"
			}
			return m, nil

		case msg.String() == "up" || msg.String() == "k":
			if m.view == "list" && m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case msg.String() == "down" || msg.String() == "j":
			if m.view == "list" && m.cursor < len(m.workflows)-1 {
				m.cursor++
			}
			return m, nil
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press r to retry, q to quit.\n", m.err)
	}

	switch m.view {
	case "detail":
		return m.detailView()
	default:
		return m.listView()
	}
}

func (m Model) listView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("⚡ Teamwork Dashboard"))
	b.WriteString("\n\n")

	if len(m.workflows) == 0 {
		b.WriteString("  No workflows found.\n")
		b.WriteString(helpStyle.Render("  r: refresh • q: quit"))
		return b.String()
	}

	// Column widths.
	const (
		colID      = 36
		colType    = 12
		colStatus  = 10
		colStep    = 6
		colRole    = 16
		colUpdated = 20
	)

	header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s",
		colID, "ID",
		colType, "Type",
		colStatus, "Status",
		colStep, "Step",
		colRole, "Role",
		colUpdated, "Updated",
	)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	for i, w := range m.workflows {
		updated := w.UpdatedAt
		if len(updated) > 19 {
			updated = updated[:19]
		}

		row := fmt.Sprintf("%-*s %-*s %-*s %-*d %-*s %-*s",
			colID, truncate(w.ID, colID),
			colType, truncate(w.Type, colType),
			colStatus, truncate(w.Status, colStatus),
			colStep, w.CurrentStep,
			colRole, truncate(w.CurrentRole, colRole),
			colUpdated, truncate(updated, colUpdated),
		)

		if i == m.cursor {
			b.WriteString(selectedStyle.Render(row))
		} else {
			b.WriteString(normalStyle.Render(row))
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("  ↑/↓,j/k: navigate • enter: detail • r: refresh • q: quit"))

	return b.String()
}

func (m Model) detailView() string {
	w := m.selected
	if w == nil {
		return "No workflow selected."
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("⚡ Workflow: %s", w.ID)))
	b.WriteString("\n\n")

	// Info section.
	info := strings.Builder{}
	info.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Type:"), w.Type))
	info.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Status:"), styledStatus(w.Status)))
	info.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Goal:"), w.Goal))
	if w.Branch != "" {
		info.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Branch:"), w.Branch))
	}
	if w.PullRequest > 0 {
		info.WriteString(fmt.Sprintf("%s #%d\n", labelStyle.Render("PR:"), w.PullRequest))
	}
	info.WriteString(fmt.Sprintf("%s %d  %s %s\n",
		labelStyle.Render("Step:"), w.CurrentStep,
		labelStyle.Render("Role:"), w.CurrentRole))
	info.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Created:"), w.CreatedAt))
	info.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Updated:"), w.UpdatedAt))

	b.WriteString(detailBorderStyle.Render(info.String()))
	b.WriteString("\n\n")

	// Step history.
	if len(w.Steps) > 0 {
		b.WriteString(labelStyle.Render("Step History"))
		b.WriteString("\n")

		for _, s := range w.Steps {
			completed := s.Completed
			if completed == "" {
				completed = "(in progress)"
			}
			gate := ""
			if s.QualityGate != "" {
				gate = fmt.Sprintf(" [%s]", s.QualityGate)
			}
			b.WriteString(fmt.Sprintf("  %d. %-16s %-30s %s%s\n",
				s.Step,
				truncate(s.Role, 16),
				truncate(s.Action, 30),
				truncate(completed, 20),
				gate,
			))
		}
		b.WriteString("\n")
	}

	// Blockers.
	if len(w.Blockers) > 0 {
		b.WriteString(statusBlockedStyle.Render("⚠ Blockers"))
		b.WriteString("\n")
		for _, bl := range w.Blockers {
			b.WriteString(fmt.Sprintf("  • %s (raised by %s at %s)\n",
				bl.Reason, bl.RaisedBy, bl.RaisedAt))
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("  esc: back • r: refresh • q: quit"))

	return b.String()
}

func (m Model) findWorkflow(id string) *state.WorkflowState {
	for _, w := range m.workflows {
		if w.ID == id {
			return w
		}
	}
	return nil
}

func styledStatus(s string) string {
	switch s {
	case state.StatusActive:
		return statusActiveStyle.Render(s)
	case state.StatusBlocked:
		return statusBlockedStyle.Render(s)
	case state.StatusCompleted:
		return statusCompletedStyle.Render(s)
	default:
		return s
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// Run starts the interactive TUI dashboard.
func Run(engine *workflow.Engine) error {
	p := tea.NewProgram(initialModel(engine), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
