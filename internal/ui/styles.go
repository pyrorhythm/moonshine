package ui

import "github.com/charmbracelet/lipgloss"

var (
	styleBrand   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	styleWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styleMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleAdd     = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	styleRemove  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleChange  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleName    = lipgloss.NewStyle().Bold(true)
	styleVersion = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
)
