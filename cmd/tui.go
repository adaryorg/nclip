/*
MIT License

Copyright (c) 2025 Yuval Adar <adary@adary.org>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

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

	// Configure program options based on configuration
	options := []tea.ProgramOption{tea.WithAltScreen()}
	
	// Add mouse support if enabled in configuration
	if cfg.Mouse.Enable {
		options = append(options, tea.WithMouseCellMotion())
	}

	p := tea.NewProgram(model, options...)

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
