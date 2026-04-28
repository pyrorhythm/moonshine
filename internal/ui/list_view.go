package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PackageEntry is one installed package shown in the packages TUI.
type PackageEntry struct {
	Name    string
	Version string
	Backend string
	Managed bool // whether the package is currently in moonfile
}

// PackagesResult is returned by RunPackagesList.
type PackagesResult struct {
	// Added are entries whose managed state changed false → true.
	Added []PackageEntry
	// Removed are entries whose managed state changed true → false.
	Removed []PackageEntry
	// Upgrade is set when the user requests an upgrade for a specific package.
	Upgrade *PackageEntry
}

// RunPackagesList presents an interactive TUI listing all installed packages.
// Returns the changes the user made plus an optional upgrade request.
func RunPackagesList(entries []PackageEntry) (PackagesResult, error) {
	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = pkgItem{PackageEntry: e, managed: e.Managed}
	}

	managed, unmanaged := 0, 0
	for _, e := range entries {
		if e.Managed {
			managed++
		} else {
			unmanaged++
		}
	}
	stats := fmt.Sprintf("%d managed · %d unmanaged", managed, unmanaged)

	l := list.New(items, pkgDelegate{}, 76, 20)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	p := tea.NewProgram(packagesModel{list: l, stats: stats}, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return PackagesResult{}, err
	}
	pm, ok := final.(packagesModel)
	if !ok {
		return PackagesResult{}, nil
	}
	return pm.result(), nil
}

type pkgItem struct {
	PackageEntry

	managed bool
}

func (i pkgItem) FilterValue() string { return i.Name + " " + i.Backend }

type pkgDelegate struct{}

func (d pkgDelegate) Height() int                             { return 1 }
func (d pkgDelegate) Spacing() int                            { return 0 }
func (d pkgDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d pkgDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	pi, ok := item.(pkgItem)
	if !ok {
		return
	}

	cursor := "  "
	nameStyle := lipgloss.NewStyle()
	if index == m.Index() {
		cursor = styleBrand.Render("> ")
		nameStyle = nameStyle.Bold(true)
	}

	ver := styleMuted.Render(pi.Version)
	be := styleMuted.Render("[" + pi.Backend + "]")
	var status string
	if pi.managed {
		status = styleSuccess.Render("✓")
	} else {
		status = styleMuted.Render("·")
	}

	fmt.Fprintf(w, "%s%s %-12s %-8s %s",
		cursor,
		lipgloss.NewStyle().Width(26).Render(nameStyle.Render(pi.Name)),
		ver, be, status,
	)
}

// packagesModel is the bubbletea model for the packages list TUI.
type packagesModel struct {
	list    list.Model
	stats   string
	upgrade *pkgItem
	done    bool
}

func (m packagesModel) Init() tea.Cmd { return nil }

func (m packagesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 5) // \n + title + \n\n + help + \n
		return m, nil

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "a":
			if sel, ok := m.list.SelectedItem().(pkgItem); ok && !sel.managed {
				sel.managed = true
				cmd := m.list.SetItem(m.list.Index(), sel)
				return m, cmd
			}
			return m, nil
		case "r":
			if sel, ok := m.list.SelectedItem().(pkgItem); ok && sel.managed {
				sel.managed = false
				cmd := m.list.SetItem(m.list.Index(), sel)
				return m, cmd
			}
			return m, nil
		case "u":
			if sel, ok := m.list.SelectedItem().(pkgItem); ok {
				cp := sel
				m.upgrade = &cp
			}
			m.done = true
			return m, tea.Quit
		case "q", "ctrl+c":
			m.done = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m packagesModel) View() string {
	if m.done {
		return ""
	}
	title := styleBrand.Render("installed packages") + "  " + styleMuted.Render(m.stats)
	help := styleMuted.Render("  a add · r remove · u upgrade · / filter · q quit")
	return "\n" + title + "\n\n" + m.list.View() + "\n" + help + "\n"
}

func (m packagesModel) result() PackagesResult {
	var res PackagesResult
	if m.upgrade != nil {
		e := m.upgrade.PackageEntry
		res.Upgrade = &e
	}
	for _, item := range m.list.Items() {
		pi, ok := item.(pkgItem)
		if !ok {
			continue
		}
		switch {
		case pi.managed && !pi.Managed:
			res.Added = append(res.Added, pi.PackageEntry)
		case !pi.managed && pi.Managed:
			res.Removed = append(res.Removed, pi.PackageEntry)
		}
	}
	return res
}
