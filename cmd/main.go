package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/adaryorg/nclip/internal/config"
	"github.com/adaryorg/nclip/internal/security"
	"github.com/adaryorg/nclip/internal/storage"
)

func main() {
	// Define command line flags
	removeSecurityInfo := flag.Bool("remove-security-information", false, "Clear all stored security hash information and start fresh")
	help := flag.Bool("help", false, "Show help information")
	flag.Parse()

	// Show help if requested
	if *help {
		showHelp()
		return
	}

	// Handle security information removal
	if *removeSecurityInfo {
		err := clearSecurityInformation()
		if err != nil {
			log.Fatalf("Failed to clear security information: %v", err)
		}
		fmt.Println("✅ Security hash information cleared successfully")
		fmt.Println("All previously blocked security content will now be collected again")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	store, err := storage.New(cfg.Database.MaxEntries)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	startTUI(store, cfg)
}

func showHelp() {
	fmt.Println("NClip - Terminal Clipboard Manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  nclip                              Start the TUI clipboard manager")
	fmt.Println("  nclip --remove-security-information Clear all stored security hash data")
	fmt.Println("  nclip --help                       Show this help message")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --remove-security-information     Removes all stored security hashes.")
	fmt.Println("                                     This clears the history of content you've")
	fmt.Println("                                     chosen to block via security warnings.")
	fmt.Println("                                     Useful for starting fresh or if the security")
	fmt.Println("                                     detection is too aggressive.")
	fmt.Println()
	fmt.Println("For more information, see the README.md file.")
}

func clearSecurityInformation() error {
	// Initialize hash store and clear it
	hashStore, err := security.NewHashStore()
	if err != nil {
		return fmt.Errorf("failed to open security hash store: %w", err)
	}
	defer hashStore.Close()

	// Get statistics before clearing
	stats, err := hashStore.GetStats()
	if err == nil {
		if totalHashes, ok := stats["total_hashes"].(int); ok {
			fmt.Printf("Removing %d stored security hashes...\n", totalHashes)
		}
	}

	// Get all hashes and remove them
	allHashes, err := hashStore.GetAllHashes("")
	if err != nil {
		return fmt.Errorf("failed to get security hashes: %w", err)
	}

	for _, hashEntry := range allHashes {
		err := hashStore.RemoveHash(hashEntry.Hash)
		if err != nil {
			return fmt.Errorf("failed to remove hash %s: %w", hashEntry.Hash[:8], err)
		}
	}

	return nil
}
