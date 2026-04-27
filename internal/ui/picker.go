package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"pyrorhythm.dev/moonshine/pkg/backend"
)

// PickSearchResult presents an interactive list of SearchResults and returns
// the one the user selects, or nil if the user aborts (q / ctrl-c / esc).
func PickSearchResult(results []backend.SearchResult) *backend.SearchResult {
	items := make([]list.Item, len(results))
	for i, r := range results {
		items[i] = searchItem{r}
	}

	const listHeight = 14
	delegate := searchDelegate{}
	l := list.New(items, delegate, 60, listHeight)
	l.Title = "Select a package to install"
	l.Styles.Title = lipgloss.
		NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212"))

	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)

	m := pickerModel{list: l}
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return nil
	}
	final, ok := result.(pickerModel)
	if !ok {
		return nil
	}
	return final.chosen
}

// searchItem wraps backend.SearchResult to implement list.Item.
type searchItem struct {
	r backend.SearchResult
}

func (i searchItem) FilterValue() string { return i.r.Name }

// searchDelegate renders each list row.
type searchDelegate struct{}

func (d searchDelegate) Height() int                             { return 1 }
func (d searchDelegate) Spacing() int                            { return 0 }
func (d searchDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d searchDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	si, ok := item.(searchItem)
	if !ok {
		return
	}
	r := si.r

	cursor := "  "
	nameStyle := lipgloss.NewStyle()
	if index == m.Index() {
		cursor = styleAdd.Render("> ")
		nameStyle = lipgloss.NewStyle().Bold(true)
	}

	ver := ""
	if r.Version != "" {
		ver = styleMuted.Render("@" + r.Version)
	}
	backend := styleMuted.Render("[" + r.Backend + "]")
	desc := ""
	if r.Description != "" {
		maxDesc := 35
		rawdesc := r.Description
		if len(rawdesc) > maxDesc {
			rawdesc = rawdesc[:maxDesc] + "…"
		}
		desc = styleMuted.Render("  " + rawdesc)
	}

	fmt.Fprintf(w, "%s%s%s %s%s\n", cursor, nameStyle.Render(r.Name), ver, backend, desc)
}

// pickerModel is the bubbletea model for the picker.
type pickerModel struct {
	list   list.Model
	chosen *backend.SearchResult
	done   bool
}

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(searchItem); ok {
				m.chosen = new(item.r)
			}
			m.done = true
			return m, tea.Quit
		case "q", "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m pickerModel) View() string {
	if m.done {
		return ""
	}
	return "\n" + m.list.View()
}
