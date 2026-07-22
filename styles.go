package main

import "charm.land/lipgloss/v2"

// Styles shared across the picker and terminal screens.
var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	dimStyle      = lipgloss.NewStyle().Faint(true)
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan
	selectedStyle = lipgloss.NewStyle().Bold(true)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	headerStyle   = lipgloss.NewStyle().Reverse(true)
	sentStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan
	noticeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
)
