package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"pyrorhythm.dev/moonshine/internal/reconciler"
)

// Message types for parallel apply progress.
type PkgStartMsg struct{ Name string }
type PkgLogMsg struct {
	Name, Line string
}
type PkgDoneMsg struct {
	Name string
	Err  error
}
type applyAllDoneMsg struct{}

// ApplyReporter implements reconciler.ProgressReporter by forwarding events to a bubbletea program.
type ApplyReporter struct {
	Send func(tea.Msg)
}

func (r *ApplyReporter) OnStart(pkg string)           { r.Send(PkgStartMsg{Name: pkg}) }
func (r *ApplyReporter) OnLog(pkg, line string)       { r.Send(PkgLogMsg{Name: pkg, Line: line}) }
func (r *ApplyReporter) OnDone(pkg string, err error) { r.Send(PkgDoneMsg{Name: pkg, Err: err}) }
func (r *ApplyReporter) OnAllDone()                   { r.Send(applyAllDoneMsg{}) }

type pkgStatus int

const (
	statusPending pkgStatus = iota
	statusRunning
	statusDone
	statusFailed
)

type pkgEntry struct {
	name   string
	status pkgStatus
	err    error
}

const maxLogLines = 6

// ApplyModel is a bubbletea model for parallel apply progress.
type ApplyModel struct {
	pkgs     []pkgEntry
	byName   map[string]int
	spinner  spinner.Model
	logs     []string
	total    int
	finished int
}

// NewApplyModel creates an ApplyModel for the given actions.
func NewApplyModel(actions []reconciler.PackageAction) ApplyModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	var pkgs []pkgEntry
	byName := make(map[string]int)
	for _, a := range actions {
		if a.Kind == reconciler.ActionNone {
			continue
		}
		n := a.DisplayName()
		byName[n] = len(pkgs)
		pkgs = append(pkgs, pkgEntry{name: n, status: statusPending})
	}
	return ApplyModel{
		pkgs:    pkgs,
		byName:  byName,
		spinner: s,
		total:   len(pkgs),
	}
}

func (m ApplyModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m ApplyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case PkgStartMsg:
		if i, ok := m.byName[msg.Name]; ok {
			m.pkgs[i].status = statusRunning
		}
	case PkgLogMsg:
		line := fmt.Sprintf("[%s] %s", msg.Name, msg.Line)
		m.logs = append(m.logs, line)
		if len(m.logs) > maxLogLines {
			m.logs = m.logs[len(m.logs)-maxLogLines:]
		}
	case PkgDoneMsg:
		if i, ok := m.byName[msg.Name]; ok {
			if msg.Err != nil {
				m.pkgs[i].status = statusFailed
			} else {
				m.pkgs[i].status = statusDone
			}
			m.pkgs[i].err = msg.Err
			m.finished++
		}
	case applyAllDoneMsg:
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ApplyModel) View() string {
	var sb strings.Builder
	sb.WriteString(styleBrand.Render("moonshine") + " applying…\n\n")

	for _, e := range m.pkgs {
		var icon string
		switch e.status {
		case statusPending:
			icon = styleMuted.Render("·")
		case statusRunning:
			icon = m.spinner.View()
		case statusDone:
			icon = styleSuccess.Render("✓")
		case statusFailed:
			icon = styleError.Render("✗")
		}
		line := fmt.Sprintf("  %s %s", icon, styleName.Render(e.name))
		if e.err != nil {
			line += " " + styleError.Render(e.err.Error())
		}
		sb.WriteString(line + "\n")
	}

	if m.total > 0 {
		sb.WriteString("\n")
		filled := 0
		if m.total > 0 {
			filled = m.finished * 20 / m.total
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", 20-filled)
		pct := m.finished * 100 / m.total
		sb.WriteString(fmt.Sprintf("  [%s] %d%%\n", styleMuted.Render(bar), pct))
	}

	if len(m.logs) > 0 {
		sb.WriteString("\n")
		for _, l := range m.logs {
			sb.WriteString(styleMuted.Render("  "+l) + "\n")
		}
	}

	return sb.String()
}

// RunApply creates and runs the apply TUI, calling run(reporter) in a goroutine.
// It blocks until the TUI exits and then returns any error from run.
func RunApply(actions []reconciler.PackageAction, run func(reconciler.ProgressReporter) error) error {
	model := NewApplyModel(actions)
	p := tea.NewProgram(model)
	reporter := &ApplyReporter{Send: func(msg tea.Msg) { p.Send(msg) }}

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(reporter)
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI: %w", err)
	}
	return <-errCh
}
