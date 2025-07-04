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
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/adaryorg/nclip/internal/clipboard"
	"github.com/adaryorg/nclip/internal/config"
	"github.com/adaryorg/nclip/internal/logging"
	"github.com/adaryorg/nclip/internal/security"
	"github.com/adaryorg/nclip/internal/storage"
)

func main() {
	cfg, err := config.LoadDaemonConfig()
	if err != nil {
		log.Fatalf("Failed to load daemon configuration: %v", err)
	}

	// Initialize logging with configured level
	logging.SetLevel(cfg.Logging.Level)
	logging.Info("Starting NClip daemon with log level: %s", cfg.Logging.Level)

	store, err := storage.New(cfg.Database.MaxEntries)
	if err != nil {
		logging.Error("Failed to initialize storage: %v", err)
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	monitor := clipboard.NewMonitorWithSecurity(
		func(content string) {
			if err := store.Add(content); err != nil {
				logging.Error("Failed to store clipboard content: %v", err)
			}
		},
		func(imageData []byte, description string) {
			if err := store.AddImage(imageData, description); err != nil {
				logging.Error("Failed to store clipboard image: %v", err)
			}
		},
		func(content string, threats []security.SecurityThreat) {
			// Security content detected - just log for awareness
			if len(threats) > 0 {
				threat := security.GetHighestThreat(threats)
				if threat != nil {
					if security.IsHighRiskThreat(threats) {
						logging.Warn("SECURITY: High-risk %s content detected (%.0f%% confidence): %s - stored with warning indicator",
							threat.Type, threat.Confidence*100, threat.Reason)
					} else {
						logging.Info("SECURITY: Medium-risk %s content detected (%.0f%% confidence): %s - stored with caution indicator",
							threat.Type, threat.Confidence*100, threat.Reason)
					}
				}
			}
		},
	)
	defer monitor.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	if err := monitor.Start(ctx); err != nil && err != context.Canceled {
		logging.Error("Monitor failed: %v", err)
		log.Fatalf("Monitor failed: %v", err)
	}
}
