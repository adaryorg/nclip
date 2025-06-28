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
	deduplicate := flag.Bool("deduplicate", false, "Remove duplicate entries from clipboard history database")
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
		fmt.Println("[OK] Security hash information cleared successfully")
		fmt.Println("All previously blocked security content will now be collected again")
		return
	}

	// Handle database deduplication
	if *deduplicate {
		err := deduplicateDatabase()
		if err != nil {
			log.Fatalf("Failed to deduplicate database: %v", err)
		}
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
	fmt.Println("  nclip --deduplicate                Remove duplicate entries from clipboard history")
	fmt.Println("  nclip --help                       Show this help message")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --remove-security-information     Removes all stored security hashes.")
	fmt.Println("                                     This clears the history of content you've")
	fmt.Println("                                     chosen to block via security warnings.")
	fmt.Println("                                     Useful for starting fresh or if the security")
	fmt.Println("                                     detection is too aggressive.")
	fmt.Println()
	fmt.Println("  --deduplicate                      Removes duplicate entries from the clipboard")
	fmt.Println("                                     history database. Keeps the most recent copy")
	fmt.Println("                                     of each unique item. Useful for cleaning up")
	fmt.Println("                                     databases that accumulated duplicates before")
	fmt.Println("                                     the automatic deduplication feature was added.")
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

func deduplicateDatabase() error {
	// Load configuration to get max entries setting
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize storage
	store, err := storage.New(cfg.Database.MaxEntries)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer store.Close()

	// Get initial count
	initialItems := store.GetAll()
	initialCount := len(initialItems)

	fmt.Printf("[INFO] Current database contains %d entries\n", initialCount)

	if initialCount == 0 {
		fmt.Println("[INFO] Database is empty, nothing to deduplicate")
		return nil
	}

	fmt.Println("[INFO] Scanning for duplicate entries...")

	// Run deduplication
	removedCount, err := store.DeduplicateExisting()
	if err != nil {
		return fmt.Errorf("failed to deduplicate database: %w", err)
	}

	// Get final count
	finalItems := store.GetAll()
	finalCount := len(finalItems)

	// Display results
	if removedCount == 0 {
		fmt.Println("[OK] No duplicate entries found! Database is already clean.")
	} else {
		fmt.Printf("[OK] Successfully removed %d duplicate entries\n", removedCount)
		fmt.Printf("[INFO] Database reduced from %d to %d entries\n", initialCount, finalCount)

		// Calculate space savings percentage
		if initialCount > 0 {
			percentage := float64(removedCount) / float64(initialCount) * 100
			fmt.Printf("[INFO] Space savings: %.1f%%\n", percentage)
		}
	}

	return nil
}
