// Updated main.go file to use pointer receivers for all model methods.

package main

import "github.com/charmbracelet/bubbletea"

// Model struct definition

// Updated Init method to use pointer receiver.
func (m *model) Init() tea.Cmd {
    // implementation
}

// Updated DoMath method to use pointer receiver.
func (m *model) DoMath(t, x, y float64) (int, int, int) {
    // implementation
}

// Updated Update method to use pointer receiver.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // implementation
}

// Updated View method to use pointer receiver.
func (m *model) View() tea.View {
    // implementation
}