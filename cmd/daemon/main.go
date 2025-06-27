package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/adaryorg/nclip/internal/clipboard"
	"github.com/adaryorg/nclip/internal/config"
	"github.com/adaryorg/nclip/internal/security"
	"github.com/adaryorg/nclip/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	store, err := storage.New(cfg.Database.MaxEntries)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	monitor := clipboard.NewMonitorWithSecurity(
		func(content string) {
			if err := store.Add(content); err != nil {
				log.Printf("Failed to store clipboard content: %v", err)
			}
		},
		func(imageData []byte, description string) {
			if err := store.AddImage(imageData, description); err != nil {
				log.Printf("Failed to store clipboard image: %v", err)
			}
		},
		func(content string, threats []security.SecurityThreat) {
			// Security content detected - just log for awareness
			if len(threats) > 0 {
				threat := security.GetHighestThreat(threats)
				if threat != nil {
					if security.IsHighRiskThreat(threats) {
						log.Printf("SECURITY: High-risk %s content detected (%.0f%% confidence): %s - stored with warning indicator",
							threat.Type, threat.Confidence*100, threat.Reason)
					} else {
						log.Printf("SECURITY: Medium-risk %s content detected (%.0f%% confidence): %s - stored with caution indicator",
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
		log.Fatalf("Monitor failed: %v", err)
	}
}
