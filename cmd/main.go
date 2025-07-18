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
	"os"

	"github.com/adaryorg/nclip/internal/config"
	"github.com/adaryorg/nclip/internal/security"
	"github.com/adaryorg/nclip/internal/storage"
	"github.com/adaryorg/nclip/internal/version"
)

func main() {
	// Define command line flags
	removeSecurityInfo := flag.Bool("remove-security-information", false, "Clear all stored security hash information and start fresh")
	deduplicate := flag.Bool("deduplicate", false, "Remove duplicate entries from clipboard history database")
	deduplicateShort := flag.Bool("d", false, "Remove duplicate entries from clipboard history database")
	prune := flag.Bool("prune", false, "Remove entries with no data or single character data from database")
	pruneShort := flag.Bool("p", false, "Remove entries with no data or single character data from database")
	rescanSecurity := flag.Bool("rescan-security", false, "Re-scan all clipboard entries with updated security detection")
	rescanSecurityShort := flag.Bool("r", false, "Re-scan all clipboard entries with updated security detection")
	basicTerminal := flag.Bool("basic-terminal", false, "Disable advanced terminal features (Unicode symbols, colors)")
	basicTerminalShort := flag.Bool("b", false, "Disable advanced terminal features (Unicode symbols, colors)")
	themeFile := flag.String("theme", "", "Use custom theme file instead of default theme.toml")
	themeFileShort := flag.String("t", "", "Use custom theme file instead of default theme.toml")
	help := flag.Bool("help", false, "Show help information")
	helpShort := flag.Bool("h", false, "Show help information")
	versionFlag := flag.Bool("version", false, "Display version and build information")
	versionShort := flag.Bool("v", false, "Display version and build information")
	flag.Parse()

	// Show version if requested
	if *versionFlag || *versionShort {
		fmt.Printf("nclip version %s | %s (%s)\n",
			version.Version,
			version.BuildTime,
			version.CommitHash,
		)
		os.Exit(0)
	}

	// Show help if requested
	if *help || *helpShort {
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
	if *deduplicate || *deduplicateShort {
		err := deduplicateDatabase()
		if err != nil {
			log.Fatalf("Failed to deduplicate database: %v", err)
		}
		return
	}

	// Handle database pruning
	if *prune || *pruneShort {
		err := pruneDatabase()
		if err != nil {
			log.Fatalf("Failed to prune database: %v", err)
		}
		return
	}

	// Handle security rescan
	if *rescanSecurity || *rescanSecurityShort {
		err := rescanSecurityThreats()
		if err != nil {
			log.Fatalf("Failed to rescan security threats: %v", err)
		}
		return
	}

	// Get custom theme file path if provided
	customThemeFile := *themeFile
	if customThemeFile == "" {
		customThemeFile = *themeFileShort
	}
	
	cfg, err := config.LoadWithCustomTheme(customThemeFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	store, err := storage.New(cfg.Database.MaxEntries)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	startTUI(store, cfg, *basicTerminal || *basicTerminalShort)
}

func showHelp() {
	fmt.Println("NClip - Terminal Clipboard Manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  nclip                              Start the TUI clipboard manager")
	fmt.Println("  nclip --remove-security-information Clear all stored security hash data")
	fmt.Println("  nclip --deduplicate, -d            Remove duplicate entries from clipboard history")
	fmt.Println("  nclip --prune, -p                  Remove entries with no data or single character data")
	fmt.Println("  nclip --rescan-security, -r        Re-scan all entries with updated security detection")
	fmt.Println("  nclip --basic-terminal, -b         Disable advanced terminal features")
	fmt.Println("  nclip --theme FILE, -t FILE        Use custom theme file instead of default")
	fmt.Println("  nclip --version, -v                Display version and build information")
	fmt.Println("  nclip --help, -h                   Show this help message")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --remove-security-information     Removes all stored security hashes.")
	fmt.Println("                                     This clears the history of content you've")
	fmt.Println("                                     chosen to block via security warnings.")
	fmt.Println("                                     Useful for starting fresh or if the security")
	fmt.Println("                                     detection is too aggressive.")
	fmt.Println()
	fmt.Println("  --deduplicate, -d                  Removes duplicate entries from the clipboard")
	fmt.Println("                                     history database. Keeps the most recent copy")
	fmt.Println("                                     of each unique item. Useful for cleaning up")
	fmt.Println("                                     databases that accumulated duplicates before")
	fmt.Println("                                     the automatic deduplication feature was added.")
	fmt.Println()
	fmt.Println("  --prune, -p                        Removes entries with no data or single")
	fmt.Println("                                     character data from the clipboard history")
	fmt.Println("                                     database. Helps clean up accidentally copied")
	fmt.Println("                                     empty strings or single characters that")
	fmt.Println("                                     clutter the clipboard history.")
	fmt.Println()
	fmt.Println("  --rescan-security, -r              Re-analyzes all clipboard entries with the")
	fmt.Println("                                     current security detection algorithms. This")
	fmt.Println("                                     updates threat levels and can reduce false")
	fmt.Println("                                     positives after security improvements.")
	fmt.Println()
	fmt.Println("  --basic-terminal, -b               Disables advanced terminal features like")
	fmt.Println("                                     Unicode symbols and colors. Use this flag")
	fmt.Println("                                     when working with old terminals or if you")
	fmt.Println("                                     experience display issues.")
	fmt.Println()
	fmt.Println("  --theme FILE, -t FILE              Use a custom theme file instead of the default")
	fmt.Println("                                     ~/.config/nclip/theme.toml. The file must be")
	fmt.Println("                                     a valid TOML file with theme configuration.")
	fmt.Println("                                     Can be an absolute path or relative to current")
	fmt.Println("                                     directory. See THEMING.md for documentation.")
	fmt.Println()
	fmt.Println("  --version, -v                      Shows the version information including")
	fmt.Println("                                     git tag, build time, and commit hash.")
	fmt.Println()
	fmt.Println("  --help, -h                         Shows this help message.")
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

func pruneDatabase() error {
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
		fmt.Println("[INFO] Database is empty, nothing to prune")
		return nil
	}

	fmt.Println("[INFO] Scanning for entries to prune...")
	fmt.Println("[INFO] Will remove entries with no data or single character data")

	// Run pruning (prune both empty data and single character entries)
	removedCount, err := store.PruneDatabase(true, true)
	if err != nil {
		return fmt.Errorf("failed to prune database: %w", err)
	}

	// Get final count
	finalItems := store.GetAll()
	finalCount := len(finalItems)

	// Display results
	if removedCount == 0 {
		fmt.Println("[OK] No entries found to prune! Database is already clean.")
	} else {
		fmt.Printf("[OK] Successfully removed %d entries\n", removedCount)
		fmt.Printf("[INFO] Database reduced from %d to %d entries\n", initialCount, finalCount)

		// Calculate space savings percentage
		if initialCount > 0 {
			percentage := float64(removedCount) / float64(initialCount) * 100
			fmt.Printf("[INFO] Space savings: %.1f%%\n", percentage)
		}
	}

	return nil
}

func rescanSecurityThreats() error {
	// Load configuration
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

	fmt.Println("[INFO] Re-scanning all clipboard entries with updated security detection...")
	fmt.Println("[INFO] This may take a moment for large databases...")

	// Run the security rescan
	stats, err := store.RescanSecurityThreats()
	if err != nil {
		return fmt.Errorf("failed to rescan security threats: %w", err)
	}

	// Display comprehensive results
	fmt.Println()
	fmt.Println("=== SECURITY RESCAN RESULTS ===")
	fmt.Printf("Total items in database: %d\n", stats["total_items"])
	fmt.Printf("Text items scanned: %d\n", stats["items_scanned"])
	fmt.Println()

	// Show threat level changes
	fmt.Println("--- THREAT LEVEL DISTRIBUTION ---")
	fmt.Printf("Before: None=%d, Low=%d, Medium=%d, High=%d (Total threats: %d)\n", 
		stats["none_before"], stats["low_before"], stats["medium_before"], stats["high_before"], stats["threats_before"])
	fmt.Printf("After:  None=%d, Low=%d, Medium=%d, High=%d (Total threats: %d)\n", 
		stats["none_after"], stats["low_after"], stats["medium_after"], stats["high_after"], stats["threats_after"])
	fmt.Println()

	// Show change summary
	fmt.Println("--- CHANGES SUMMARY ---")
	fmt.Printf("Items unchanged: %d\n", stats["unchanged"])
	fmt.Printf("Items downgraded (less threatening): %d\n", stats["downgraded"])
	fmt.Printf("Items upgraded (more threatening): %d\n", stats["upgraded"])
	fmt.Println()

	// Calculate and show improvement statistics
	threatsReduced := stats["threats_before"] - stats["threats_after"]
	if threatsReduced > 0 {
		fmt.Printf("✅ IMPROVEMENT: %d items are no longer flagged as threats\n", threatsReduced)
		if stats["threats_before"] > 0 {
			percentage := float64(threatsReduced) / float64(stats["threats_before"]) * 100
			fmt.Printf("✅ False positive reduction: %.1f%%\n", percentage)
		}
	} else if threatsReduced < 0 {
		fmt.Printf("⚠️  %d additional items are now flagged as threats\n", -threatsReduced)
	} else {
		fmt.Println("ℹ️  No change in total number of threats detected")
	}

	// Show specific improvements
	if stats["downgraded"] > 0 {
		fmt.Printf("✅ %d items had their threat level reduced\n", stats["downgraded"])
	}
	if stats["upgraded"] > 0 {
		fmt.Printf("⚠️  %d items had their threat level increased\n", stats["upgraded"])
	}

	fmt.Println()
	fmt.Println("[OK] Security rescan completed successfully!")
	fmt.Println("💡 The improved detection should show fewer false positives for:")
	fmt.Println("   • Regular text and sentences")
	fmt.Println("   • Simple URLs without query parameters")
	fmt.Println("   • Code snippets and technical content")
	fmt.Println("   • Short strings and common words")

	return nil
}
