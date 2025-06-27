package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adaryorg/nclip/internal/config"
	"github.com/adaryorg/nclip/internal/storage"
	"github.com/adaryorg/nclip/internal/ui"
)

func startTUI(store *storage.Storage, cfg *config.Config) {
	model := ui.NewModel(store, cfg)

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
