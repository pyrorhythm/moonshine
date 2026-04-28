package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	fmt.Printf("Height of empty string: %d\n", lipgloss.Height(""))
	fmt.Printf("Height of newline: %d\n", lipgloss.Height("\n"))
	
	style := lipgloss.NewStyle().Height(0)
	out := style.Render("hello")
	fmt.Printf("Height(0) render: %q (height=%d)\n", out, lipgloss.Height(out))
}
