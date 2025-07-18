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
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adaryorg/nclip/internal/clipboard"
	"github.com/adaryorg/nclip/internal/config"
	"github.com/adaryorg/nclip/internal/logging"
	"github.com/adaryorg/nclip/internal/security"
	"github.com/adaryorg/nclip/internal/storage"
	"github.com/adaryorg/nclip/internal/version"
)

func main() {
	// Define command line flags
	versionFlag := flag.Bool("version", false, "Display version and build information")
	versionShort := flag.Bool("v", false, "Display version and build information")
	flag.Parse()

	// Show version if requested
	if *versionFlag || *versionShort {
		fmt.Printf("nclipd version %s | %s (%s)\n",
			version.Version,
			version.BuildTime,
			version.CommitHash,
		)
		os.Exit(0)
	}

	cfg, err := config.LoadDaemonConfig()
	if err != nil {
		log.Fatalf("Failed to load daemon configuration: %v", err)
	}

	// Initialize logging with file rotation and configured level
	err = logging.InitLogger(
		cfg.Logging.LogFile,
		cfg.Logging.Level,
		cfg.Logging.MaxAge,
		cfg.Logging.MaxSize,
		cfg.Logging.MaxBackups,
	)
	if err != nil {
		log.Fatalf("Failed to initialize logging: %v", err)
	}
	
	logging.Info("Starting NClip daemon with log level: %s", cfg.Logging.Level)
	logging.Info("Log file: %s", cfg.Logging.LogFile)

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

	// Start maintenance tasks
	if cfg.Maintenance.AutoDedupe {
		go startMaintenanceTask(ctx, store, "deduplication", time.Duration(cfg.Maintenance.DedupeInterval)*time.Minute, func() {
			logging.Info("Running automatic deduplication...")
			if removedCount, err := store.DeduplicateExisting(); err != nil {
				logging.Error("Automatic deduplication failed: %v", err)
			} else if removedCount > 0 {
				logging.Info("Automatic deduplication removed %d duplicates", removedCount)
			}
		})
	}

	if cfg.Maintenance.AutoPrune {
		go startMaintenanceTask(ctx, store, "pruning", time.Duration(cfg.Maintenance.PruneInterval)*time.Minute, func() {
			logging.Info("Running automatic database pruning...")
			if removedCount, err := store.PruneDatabase(cfg.Maintenance.PruneEmptyData, cfg.Maintenance.PruneSingleChar); err != nil {
				logging.Error("Automatic pruning failed: %v", err)
			} else if removedCount > 0 {
				logging.Info("Automatic pruning removed %d entries", removedCount)
			}
		})
	}

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

// startMaintenanceTask runs a maintenance task at regular intervals
func startMaintenanceTask(ctx context.Context, store *storage.Storage, taskName string, interval time.Duration, task func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logging.Info("Starting automatic %s task (every %v)", taskName, interval)

	for {
		select {
		case <-ctx.Done():
			logging.Info("Stopping %s maintenance task", taskName)
			return
		case <-ticker.C:
			task()
		}
	}
}
