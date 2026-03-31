// Package search provides full-text search over teamwork artifacts.
//
// It searches memory entries, handoff artifacts, ADRs, and workflow state files,
// ranking results by term frequency and returning snippets of matching context.
package search

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Result represents a single search hit.
type Result struct {
	Path    string
	Type    string // "memory", "handoff", "adr", "state"
	Score   int    // term frequency across the document
	Snippet string // up to 200 chars of surrounding context
}

// QueryOptions configures optional search filters.
type QueryOptions struct {
	Domain string // filter memory by domain tag (empty = no filter)
	Type   string // filter by artifact type: "memory"|"handoff"|"adr"|"state"|"" (all)
}

// Query searches the teamwork artifact corpus in dir for query and returns
// results sorted by score descending. Returns an error if query is empty.
func Query(dir, query string, opts QueryOptions) ([]Result, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search: query must not be empty")
	}

	terms := strings.Fields(strings.ToLower(query))
	var results []Result

	if opts.Type == "" || opts.Type == "memory" {
		mem, err := searchMemory(dir, terms, opts.Domain)
		if err != nil {
			return nil, err
		}
		results = append(results, mem...)
	}

	if opts.Type == "" || opts.Type == "handoff" {
		ho, err := searchFiles(dir, filepath.Join(".teamwork", "handoffs"), "handoff", terms)
		if err != nil {
			return nil, err
		}
		results = append(results, ho...)
	}

	if opts.Type == "" || opts.Type == "adr" {
		adrs, err := searchFiles(dir, filepath.Join("docs", "decisions"), "adr", terms)
		if err != nil {
			return nil, err
		}
		results = append(results, adrs...)
	}

	if opts.Type == "" || opts.Type == "state" {
		st, err := searchStateFiles(dir, terms)
		if err != nil {
			return nil, err
		}
		results = append(results, st...)
	}

	// Sort by score descending (simple insertion sort — corpus is small).
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Score > results[j-1].Score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}

	return results, nil
}

// memoryEntry is the minimal YAML structure we need from memory files.
type memoryEntry struct {
	ID      string   `yaml:"id"`
	Domain  []string `yaml:"domain"`
	Content string   `yaml:"content"`
	Context string   `yaml:"context"`
}

type memoryFile struct {
	Entries []memoryEntry `yaml:"entries"`
}

// searchMemory searches .teamwork/memory/*.yaml files.
func searchMemory(dir string, terms []string, domain string) ([]Result, error) {
	memDir := filepath.Join(dir, ".teamwork", "memory")
	entries, err := os.ReadDir(memDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("search: read memory dir: %w", err)
	}

	var results []Result
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		// Skip archive files.
		if strings.Contains(entry.Name(), ".archive-") {
			continue
		}

		p := filepath.Join(memDir, entry.Name())
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}

		var mf memoryFile
		if err := yaml.Unmarshal(data, &mf); err != nil {
			continue
		}

		for _, e := range mf.Entries {
			// Apply domain filter.
			if domain != "" && !containsDomain(e.Domain, domain) {
				continue
			}

			corpus := strings.ToLower(e.Content + " " + e.Context)
			score, snippet := scoreAndSnippet(corpus, terms, e.Content+" "+e.Context)
			if score > 0 {
				results = append(results, Result{
					Path:    p,
					Type:    "memory",
					Score:   score,
					Snippet: snippet,
				})
			}
		}
	}
	return results, nil
}

// searchFiles walks a directory recursively and searches file contents.
func searchFiles(dir, relPath, artifactType string, terms []string) ([]Result, error) {
	root := filepath.Join(dir, relPath)
	var results []Result

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		// For handoffs: .md files; for ADRs: .md files.
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		content := string(data)
		corpus := strings.ToLower(content)
		score, snippet := scoreAndSnippet(corpus, terms, content)
		if score > 0 {
			results = append(results, Result{
				Path:    path,
				Type:    artifactType,
				Score:   score,
				Snippet: snippet,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("search: walk %s: %w", root, err)
	}

	return results, nil
}

// stateFile is the minimal YAML structure we need from state files.
type stateFile struct {
	Goal string `yaml:"goal"`
}

// searchStateFiles searches .teamwork/state/*.yaml files by goal field.
func searchStateFiles(dir string, terms []string) ([]Result, error) {
	stateDir := filepath.Join(dir, ".teamwork", "state")
	var results []Result

	err := filepath.Walk(stateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		var sf stateFile
		if err := yaml.Unmarshal(data, &sf); err != nil {
			return nil
		}

		corpus := strings.ToLower(sf.Goal)
		score, snippet := scoreAndSnippet(corpus, terms, sf.Goal)
		if score > 0 {
			results = append(results, Result{
				Path:    path,
				Type:    "state",
				Score:   score,
				Snippet: snippet,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("search: walk state dir: %w", err)
	}

	return results, nil
}

// scoreAndSnippet counts term occurrences in corpus (lowercase) and returns a
// snippet from the original content surrounding the first match.
// corpus must be the lowercased version of content.
func scoreAndSnippet(corpus string, terms []string, content string) (int, string) {
	score := 0
	firstIdx := -1

	for _, term := range terms {
		idx := 0
		for {
			pos := strings.Index(corpus[idx:], term)
			if pos < 0 {
				break
			}
			score++
			absPos := idx + pos
			if firstIdx < 0 {
				firstIdx = absPos
			}
			idx = absPos + len(term)
		}
	}

	if score == 0 {
		return 0, ""
	}

	snippet := extractSnippet(content, firstIdx, 200)
	return score, snippet
}

// extractSnippet extracts up to maxLen chars centered around pos.
func extractSnippet(content string, pos, maxLen int) string {
	if pos < 0 || len(content) == 0 {
		if len(content) > maxLen {
			return content[:maxLen]
		}
		return content
	}

	start := pos - maxLen/4
	if start < 0 {
		start = 0
	}
	end := start + maxLen
	if end > len(content) {
		end = len(content)
	}

	snippet := content[start:end]
	// Trim to clean boundaries if possible.
	snippet = strings.TrimSpace(snippet)
	return snippet
}

// containsDomain reports whether domains contains the given domain.
func containsDomain(domains []string, domain string) bool {
	for _, d := range domains {
		if d == domain {
			return true
		}
	}
	return false
}
