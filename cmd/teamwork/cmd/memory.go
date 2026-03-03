package cmd

import (
	"fmt"
	"strings"

	"github.com/JoshLuedeman/teamwork/internal/config"
	"github.com/JoshLuedeman/teamwork/internal/memory"
	"github.com/spf13/cobra"
)

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage structured project memory",
}

var memoryAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a memory entry",
	RunE:  runMemoryAdd,
}

var memorySearchCmd = &cobra.Command{
	Use:   "search [domain]",
	Short: "Search memory entries by domain",
	Args:  cobra.ExactArgs(1),
	RunE:  runMemorySearch,
}

var memoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List memory entries",
	RunE:  runMemoryList,
}

var memorySyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync memory entries to a spoke repository",
	RunE:  runMemorySync,
}

func init() {
	memoryAddCmd.Flags().String("category", "", "Category: patterns, antipatterns, decisions, feedback (required)")
	memoryAddCmd.Flags().String("domain", "", "Comma-separated domain tags (required)")
	memoryAddCmd.Flags().String("content", "", "Memory content (required)")
	memoryAddCmd.Flags().String("source", "", "Where this was learned (e.g. PR #42)")
	memoryAddCmd.Flags().String("context", "", "Additional context")
	_ = memoryAddCmd.MarkFlagRequired("category")
	_ = memoryAddCmd.MarkFlagRequired("domain")
	_ = memoryAddCmd.MarkFlagRequired("content")

	memoryListCmd.Flags().String("category", "", "Filter by category: patterns, antipatterns, decisions, feedback")

	memorySyncCmd.Flags().String("repo", "", "Target repo name from config (required)")
	memorySyncCmd.Flags().String("domain", "", "Comma-separated domains to sync (required)")
	_ = memorySyncCmd.MarkFlagRequired("repo")
	_ = memorySyncCmd.MarkFlagRequired("domain")

	memoryCmd.AddCommand(memoryAddCmd)
	memoryCmd.AddCommand(memorySearchCmd)
	memoryCmd.AddCommand(memoryListCmd)
	memoryCmd.AddCommand(memorySyncCmd)
	rootCmd.AddCommand(memoryCmd)
}

func runMemoryAdd(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	catStr, _ := cmd.Flags().GetString("category")
	domainStr, _ := cmd.Flags().GetString("domain")
	content, _ := cmd.Flags().GetString("content")
	source, _ := cmd.Flags().GetString("source")
	context, _ := cmd.Flags().GetString("context")

	cat, err := parseCategory(catStr)
	if err != nil {
		return err
	}

	domains := strings.Split(domainStr, ",")
	for i := range domains {
		domains[i] = strings.TrimSpace(domains[i])
	}

	entry := memory.Entry{
		Source:  source,
		Domain:  domains,
		Content: content,
		Context: context,
	}

	if err := memory.Add(dir, cat, entry); err != nil {
		return fmt.Errorf("adding memory entry: %w", err)
	}

	fmt.Printf("Added %s entry to %s (domains: %s)\n", cat, catStr+".yaml", domainStr)
	return nil
}

func runMemorySearch(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	entries, err := memory.Search(dir, args[0])
	if err != nil {
		return fmt.Errorf("searching memory: %w", err)
	}

	if len(entries) == 0 {
		fmt.Printf("No entries found for domain %q.\n", args[0])
		return nil
	}

	fmt.Printf("%-16s  %-12s  %-20s  %s\n", "ID", "Date", "Source", "Content")
	fmt.Println("----------------  ------------  --------------------  ----------------------------------------")
	for _, e := range entries {
		content := e.Content
		if len(content) > 40 {
			content = content[:37] + "..."
		}
		fmt.Printf("%-16s  %-12s  %-20s  %s\n", e.ID, e.Date, truncate(e.Source, 20), content)
	}

	return nil
}

func runMemoryList(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	catFilter, _ := cmd.Flags().GetString("category")

	categories := []memory.Category{memory.Patterns, memory.Antipatterns, memory.Decisions, memory.Feedback}
	if catFilter != "" {
		cat, err := parseCategory(catFilter)
		if err != nil {
			return err
		}
		categories = []memory.Category{cat}
	}

	fmt.Printf("%-16s  %-14s  %-12s  %-20s  %s\n", "ID", "Category", "Date", "Domains", "Content")
	fmt.Println("----------------  --------------  ------------  --------------------  ----------------------------------------")

	total := 0
	for _, cat := range categories {
		mf, err := memory.LoadCategory(dir, cat)
		if err != nil {
			return fmt.Errorf("loading %s: %w", cat, err)
		}
		for _, e := range mf.Entries {
			content := e.Content
			if len(content) > 40 {
				content = content[:37] + "..."
			}
			fmt.Printf("%-16s  %-14s  %-12s  %-20s  %s\n",
				e.ID, cat, e.Date, truncate(strings.Join(e.Domain, ","), 20), content)
			total++
		}
	}

	if total == 0 {
		fmt.Println("No entries found.")
	} else {
		fmt.Printf("\n%d entries total\n", total)
	}
	return nil
}

func parseCategory(s string) (memory.Category, error) {
	switch s {
	case "patterns":
		return memory.Patterns, nil
	case "antipatterns":
		return memory.Antipatterns, nil
	case "decisions":
		return memory.Decisions, nil
	case "feedback":
		return memory.Feedback, nil
	default:
		return "", fmt.Errorf("invalid category %q: must be patterns, antipatterns, decisions, or feedback", s)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func runMemorySync(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	repoName, _ := cmd.Flags().GetString("repo")
	domainStr, _ := cmd.Flags().GetString("domain")

	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	repo := cfg.GetRepo(repoName)
	if repo == nil {
		return fmt.Errorf("repo %q not found in config", repoName)
	}

	domains := strings.Split(domainStr, ",")
	synced := 0
	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		entries, err := memory.Search(dir, domain)
		if err != nil {
			return fmt.Errorf("searching domain %q: %w", domain, err)
		}

		for _, e := range entries {
			// Determine category from the entry ID prefix.
			cat := categoryFromID(e.ID)
			if cat == "" {
				continue
			}
			parsed, _ := parseCategory(cat)
			if err := memory.Add(repo.Path, parsed, e); err != nil {
				return fmt.Errorf("syncing entry %s to %s: %w", e.ID, repoName, err)
			}
			synced++
		}
	}

	fmt.Printf("Synced %d entries to %s (%s)\n", synced, repoName, repo.Path)
	return nil
}

func categoryFromID(id string) string {
	if strings.HasPrefix(id, "pattern-") {
		return "patterns"
	}
	if strings.HasPrefix(id, "antipattern-") {
		return "antipatterns"
	}
	if strings.HasPrefix(id, "decision-") {
		return "decisions"
	}
	if strings.HasPrefix(id, "feedback-") {
		return "feedback"
	}
	return ""
}
