package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// DriftResult describes the difference between installed and upstream framework files.
type DriftResult struct {
	Added    []string // files in upstream but not installed locally
	Modified []string // files whose content differs between local and upstream
	Removed  []string // files that are managed locally but no longer in upstream
}

// HasDrift reports whether any drift was detected.
func (d *DriftResult) HasDrift() bool {
	return len(d.Added) > 0 || len(d.Modified) > 0 || len(d.Removed) > 0
}

// CheckDrift downloads the latest tarball from upstream and compares framework
// files against the currently installed versions in dir. It does NOT modify
// any files on disk. Returns a DriftResult describing what would change.
func CheckDrift(dir, owner, repo, ref string) (*DriftResult, error) {
	upstreamFiles, _, err := fetchTarball(owner, repo, ref)
	if err != nil {
		return nil, fmt.Errorf("fetching tarball: %w", err)
	}

	// Apply the same language-file filter used by Install/Update so language-
	// specific files for languages not present in dir are not counted as drift.
	upstreamFiles = filterLanguageFiles(upstreamFiles, dir)

	// Build a map of upstream files for fast lookup.
	upstreamMap := make(map[string][]byte, len(upstreamFiles))
	for _, f := range upstreamFiles {
		upstreamMap[f.Path] = f.Data
	}

	// Read the manifest to know which files are framework-managed on disk.
	manifest, err := readManifest(dir)
	if err != nil {
		// No manifest: treat as untracked install; check only by comparing files.
		manifest = &Manifest{Files: make(map[string]string)}
	}

	result := &DriftResult{}

	// Check each upstream file against the on-disk version.
	for _, f := range upstreamFiles {
		localPath := filepath.Join(dir, f.Path)
		existing, readErr := os.ReadFile(localPath)
		if readErr != nil {
			// File does not exist locally - it would be added.
			result.Added = append(result.Added, f.Path)
			continue
		}
		if sha256hex(existing) != sha256hex(f.Data) {
			result.Modified = append(result.Modified, f.Path)
		}
	}

	// Check for files that are in the manifest (managed by framework) but
	// no longer present in the upstream tarball - they would be removed.
	for installedPath := range manifest.Files {
		if _, inUpstream := upstreamMap[installedPath]; !inUpstream {
			// Only flag as removed if the file still exists locally.
			if _, statErr := os.Stat(filepath.Join(dir, installedPath)); statErr == nil {
				result.Removed = append(result.Removed, installedPath)
			}
		}
	}

	// Sort for deterministic output.
	sort.Strings(result.Added)
	sort.Strings(result.Modified)
	sort.Strings(result.Removed)

	return result, nil
}
