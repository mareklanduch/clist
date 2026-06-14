package tui

import (
	"github.com/charmbracelet/lipgloss"

	"clist/internal/task"
)

// Color palette — kept local to the TUI package so the core packages
// stay free of UI dependencies.
var (
	colCyan     = lipgloss.Color("14")
	colGreen    = lipgloss.Color("10")
	colYellow   = lipgloss.Color("11")
	colRed      = lipgloss.Color("9")
	colGray     = lipgloss.Color("8")
	colWhite    = lipgloss.Color("15")
	colBlue     = lipgloss.Color("12")
	colLightRed = lipgloss.Color("203")
)

// priorityColor maps a task priority to a lipgloss color.
func priorityColor(p task.Priority) lipgloss.Color {
	switch p {
	case task.PriorityCritical:
		return colLightRed
	case task.PriorityHigh:
		return colRed
	case task.PriorityMedium:
		return colYellow
	default:
		return colBlue
	}
}
