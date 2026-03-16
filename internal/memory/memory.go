// Package memory manages structured project memory in .teamwork/memory/.
//
// Memory entries capture patterns, antipatterns, decisions, and feedback that
// persist across agent sessions, enabling institutional knowledge.
package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Entry represents a single memory entry in a category file.
type Entry struct {
	ID      string   `yaml:"id"`
	Date    string   `yaml:"date"`
	Source  string   `yaml:"source"`
	Domain  []string `yaml:"domain"`
	Content string   `yaml:"content"`
	Context string   `yaml:"context"`
}

// MemoryFile holds all entries for a single memory category.
type MemoryFile struct {
	Entries []Entry `yaml:"entries"`
}

// Index maps domains to entry IDs for fast lookup.
type Index struct {
	Domains map[string][]string `yaml:"domains"`
}

// Category represents a memory file type.
type Category string

const (
	Patterns     Category = "patterns"
	Antipatterns Category = "antipatterns"
	Decisions    Category = "decisions"
	Feedback     Category = "feedback"
)

// categoryPath returns the file path for a category YAML file.
func categoryPath(dir string, cat Category) string {
	return filepath.Join(dir, ".teamwork", "memory", string(cat)+".yaml")
}

// indexPath returns the file path for the index YAML file.
func indexPath(dir string) string {
	return filepath.Join(dir, ".teamwork", "memory", "index.yaml")
}

// LoadCategory loads a memory category file from .teamwork/memory/<category>.yaml.
func LoadCategory(dir string, cat Category) (*MemoryFile, error) {
	p := categoryPath(dir, cat)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &MemoryFile{}, nil
		}
		return nil, fmt.Errorf("reading memory file %s: %w", p, err)
	}

	var mf MemoryFile
	if err := yaml.Unmarshal(data, &mf); err != nil {
		return nil, fmt.Errorf("parsing memory file %s: %w", p, err)
	}
	return &mf, nil
}

// SaveCategory writes a memory category file to disk.
func SaveCategory(dir string, cat Category, mf *MemoryFile) error {
	p := categoryPath(dir, cat)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("creating memory directory: %w", err)
	}

	data, err := yaml.Marshal(mf)
	if err != nil {
		return fmt.Errorf("marshaling memory file: %w", err)
	}
	return os.WriteFile(p, data, 0o644)
}

// LoadIndex loads the domain index from .teamwork/memory/index.yaml.
func LoadIndex(dir string) (*Index, error) {
	p := indexPath(dir)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Index{Domains: make(map[string][]string)}, nil
		}
		return nil, fmt.Errorf("reading index file: %w", err)
	}

	var idx Index
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing index file: %w", err)
	}
	if idx.Domains == nil {
		idx.Domains = make(map[string][]string)
	}
	return &idx, nil
}

// SaveIndex writes the domain index to disk.
func SaveIndex(dir string, idx *Index) error {
	p := indexPath(dir)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("creating memory directory: %w", err)
	}

	data, err := yaml.Marshal(idx)
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}
	return os.WriteFile(p, data, 0o644)
}

// Add appends an entry to the specified category and updates the domain index.
// If the entry has no ID, one is generated automatically.
// When archiveThreshold is greater than zero, entries exceeding the threshold
// are automatically rotated to a dated archive file.
func Add(dir string, cat Category, entry Entry, archiveThreshold int) error {
	mf, err := LoadCategory(dir, cat)
	if err != nil {
		return fmt.Errorf("loading category %s: %w", cat, err)
	}

	if entry.ID == "" {
		entry.ID = NextID(cat, mf.Entries)
	}
	if entry.Date == "" {
		entry.Date = time.Now().UTC().Format("2006-01-02")
	}

	mf.Entries = append(mf.Entries, entry)

	if err := SaveCategory(dir, cat, mf); err != nil {
		return fmt.Errorf("saving category %s: %w", cat, err)
	}

	// Update the index with new domain mappings.
	idx, err := LoadIndex(dir)
	if err != nil {
		return fmt.Errorf("loading index: %w", err)
	}
	for _, domain := range entry.Domain {
		if !contains(idx.Domains[domain], entry.ID) {
			idx.Domains[domain] = append(idx.Domains[domain], entry.ID)
		}
	}
	if err := SaveIndex(dir, idx); err != nil {
		return fmt.Errorf("saving index: %w", err)
	}

	// Archive old entries when the threshold is exceeded.
	if archiveThreshold > 0 && NeedsArchive(mf, archiveThreshold) {
		if err := Archive(dir, cat, archiveThreshold); err != nil {
			return fmt.Errorf("archiving %s: %w", cat, err)
		}
	}

	return nil
}

// Search finds all entries matching the given domain across all category files.
func Search(dir string, domain string) ([]Entry, error) {
	idx, err := LoadIndex(dir)
	if err != nil {
		return nil, fmt.Errorf("loading index: %w", err)
	}

	ids, ok := idx.Domains[domain]
	if !ok {
		return nil, nil
	}

	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}

	var results []Entry
	for _, cat := range []Category{Patterns, Antipatterns, Decisions, Feedback} {
		mf, err := LoadCategory(dir, cat)
		if err != nil {
			return nil, fmt.Errorf("loading category %s: %w", cat, err)
		}
		for _, e := range mf.Entries {
			if idSet[e.ID] {
				results = append(results, e)
			}
		}
	}

	return results, nil
}

// NextID generates the next sequential ID for a category, e.g. "pattern-005".
func NextID(cat Category, entries []Entry) string {
	prefix := categoryPrefix(cat)
	maxNum := 0
	for _, e := range entries {
		if strings.HasPrefix(e.ID, prefix) {
			numStr := strings.TrimPrefix(e.ID, prefix)
			if n, err := strconv.Atoi(numStr); err == nil && n > maxNum {
				maxNum = n
			}
		}
	}
	return fmt.Sprintf("%s%03d", prefix, maxNum+1)
}

// Rebuild reconstructs the domain index from all memory category files.
func (idx *Index) Rebuild(dir string) error {
	idx.Domains = make(map[string][]string)

	for _, cat := range []Category{Patterns, Antipatterns, Decisions, Feedback} {
		mf, err := LoadCategory(dir, cat)
		if err != nil {
			return fmt.Errorf("loading category %s: %w", cat, err)
		}
		for _, e := range mf.Entries {
			for _, domain := range e.Domain {
				if !contains(idx.Domains[domain], e.ID) {
					idx.Domains[domain] = append(idx.Domains[domain], e.ID)
				}
			}
		}
	}

	// Sort domain entries for deterministic output.
	for domain := range idx.Domains {
		sort.Strings(idx.Domains[domain])
	}

	return SaveIndex(dir, idx)
}

// NeedsArchive reports whether a memory file has more entries than the given threshold.
func NeedsArchive(mf *MemoryFile, threshold int) bool {
	return len(mf.Entries) > threshold
}

// nowFunc is the time source used for archive file naming, overridden in tests.
var nowFunc = time.Now

// Archive moves entries beyond the threshold to a dated archive file,
// keeping the most recent entries in the main category file. The domain
// index is rebuilt after archiving so it reflects only the active entries.
func Archive(dir string, cat Category, threshold int) error {
	mf, err := LoadCategory(dir, cat)
	if err != nil {
		return fmt.Errorf("loading category %s: %w", cat, err)
	}

	if !NeedsArchive(mf, threshold) {
		return nil
	}

	// Keep the most recent entries (last threshold), archive the rest.
	cutoff := len(mf.Entries) - threshold
	toArchive := make([]Entry, cutoff)
	copy(toArchive, mf.Entries[:cutoff])
	toKeep := make([]Entry, threshold)
	copy(toKeep, mf.Entries[cutoff:])

	// Load or create the archive file for the current month.
	archPath := archiveFilePath(dir, cat, nowFunc())
	archFile, err := loadMemoryFile(archPath)
	if err != nil {
		return fmt.Errorf("loading archive file: %w", err)
	}
	archFile.Entries = append(archFile.Entries, toArchive...)

	if err := saveMemoryFile(archPath, archFile); err != nil {
		return fmt.Errorf("saving archive file: %w", err)
	}

	// Trim the main file to only the kept entries.
	mf.Entries = toKeep
	if err := SaveCategory(dir, cat, mf); err != nil {
		return fmt.Errorf("saving trimmed category %s: %w", cat, err)
	}

	// Rebuild the index to reflect removed entries.
	idx := &Index{}
	if err := idx.Rebuild(dir); err != nil {
		return fmt.Errorf("rebuilding index after archive: %w", err)
	}

	return nil
}

// archiveFilePath returns the path for a dated archive file.
func archiveFilePath(dir string, cat Category, t time.Time) string {
	name := fmt.Sprintf("%s.archive-%s.yaml", string(cat), t.Format("2006-01"))
	return filepath.Join(dir, ".teamwork", "memory", name)
}

// loadMemoryFile reads a MemoryFile from the given path.
// Returns an empty MemoryFile if the file does not exist.
func loadMemoryFile(path string) (*MemoryFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &MemoryFile{}, nil
		}
		return nil, fmt.Errorf("reading file %s: %w", path, err)
	}

	var mf MemoryFile
	if err := yaml.Unmarshal(data, &mf); err != nil {
		return nil, fmt.Errorf("parsing file %s: %w", path, err)
	}
	return &mf, nil
}

// saveMemoryFile writes a MemoryFile to the given path.
func saveMemoryFile(path string, mf *MemoryFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	data, err := yaml.Marshal(mf)
	if err != nil {
		return fmt.Errorf("marshaling file: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// categoryPrefix returns the ID prefix for a category (e.g. "pattern-").
func categoryPrefix(cat Category) string {
	s := string(cat)
	// Singularize: patterns → pattern, antipatterns → antipattern, etc.
	s = strings.TrimSuffix(s, "s")
	return s + "-"
}

// contains reports whether slice contains the given string.
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
