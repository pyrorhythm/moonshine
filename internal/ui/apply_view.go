package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"pyrorhythm.dev/moonshine/internal/reconciler"
)

// ProgressMsg is sent when a package action completes.
type ProgressMsg struct {
	Action reconciler.PackageAction
	Err    error
	Done   bool // true = all done
}

type applyRow struct {
	action reconciler.PackageAction
	done   bool
	err    error
}

// ApplyModel is a bubbletea model that shows live apply progress.
type ApplyModel struct {
	spinner  spinner.Model
	rows     []applyRow
	current  int
	quitting bool
	err      error
}

// NewApplyModel creates an ApplyModel for the given actions.
func NewApplyModel(actions []reconciler.PackageAction) ApplyModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	rows := make([]applyRow, 0, len(actions))
	for _, a := range actions {
		if a.Kind != reconciler.ActionNone {
			rows = append(rows, applyRow{action: a})
		}
	}
	return ApplyModel{spinner: s, rows: rows}
}

func (m ApplyModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m ApplyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ProgressMsg:
		if msg.Done {
			m.quitting = true
			return m, tea.Quit
		}
		if m.current < len(m.rows) {
			m.rows[m.current].done = true
			m.rows[m.current].err = msg.Err
			m.current++
		}
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ApplyModel) View() string {
	var sb strings.Builder
	sb.WriteString(styleBrand.Render("moonshine") + " applying…\n\n")

	for i, row := range m.rows {
		a := row.action
		name := pkgName(a)
		prefix := ""

		switch {
		case row.done && row.err == nil:
			prefix = styleSuccess.Render("✓")
		case row.done && row.err != nil:
			prefix = styleError.Render("✗")
		case i == m.current:
			prefix = m.spinner.View()
		default:
			prefix = styleMuted.Render("·")
		}

		actionStr := styleChange.Render(a.Kind.String())
		line := fmt.Sprintf("  %s %s %s", prefix, actionStr, styleName.Render(name))
		if row.err != nil {
			line += " " + styleError.Render(row.err.Error())
		}
		sb.WriteString(line + "\n")
	}

	if m.quitting {
		sb.WriteString("\n")
	}
	return sb.String()
}

func pkgName(a reconciler.PackageAction) string {
	if a.Package.Meta != nil {
		if n := a.Package.Name(); n != "" {
			return n
		}
	}
	if a.Current != nil {
		return a.Current.Name
	}
	return "?"
}
