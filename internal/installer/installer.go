// Package installer fetches Teamwork framework files from GitHub and manages
// install/update with manifest-based conflict detection.
package installer

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"time"
)

// FrameworkFiles is the list of path prefixes extracted from the tarball.
// These are overwritten on update (if unchanged by user).
var FrameworkFiles = []string{
	".github/agents/",
	".github/skills/",
	".github/instructions/",
	".github/copilot-instructions.md",
	".github/ISSUE_TEMPLATE/",
	".github/PULL_REQUEST_TEMPLATE.md",
	"docs/",
	"scripts/",
	".editorconfig",
	".pre-commit-config.yaml",
	".teamwork/config.yaml",
	"Makefile",
}

// StarterTemplates maps relative path to content for files created once on install.
// Never overwritten by update.
var StarterTemplates = map[string]string{
	"MEMORY.md": "# Project Memory\n\nThis file captures project learnings that persist across agent sessions.\n",
	"CHANGELOG.md": "# Changelog\n\nAll notable changes to this project will be documented in this file.\n\n" +
		"The format is based on [Keep a Changelog](https://keepachangelog.com/).\n",
}

// languageInstructionFiles maps instruction file paths to the marker files
// that indicate the language is used in the project. If none of the markers
// exist in the target directory, the instruction file is skipped.
var languageInstructionFiles = map[string][]string{
	".github/instructions/go.instructions.md": {"go.mod", "go.sum"},
}

// File represents a single file extracted from the tarball.
type File struct {
	Path string
	Data []byte
}

// Manifest stores SHA-256 hashes of installed framework files.
type Manifest struct {
	Version string            `json:"version"`
	Files   map[string]string `json:"files"`
}

const (
	manifestPath = ".teamwork/framework-manifest.json"
	versionPath  = ".teamwork/framework-version.txt"
)

// Install fetches framework files from upstream and writes them to dir.
func Install(dir, owner, repo, ref string) error {
	vp := filepath.Join(dir, versionPath)
	if _, err := os.Stat(vp); err == nil {
		return fmt.Errorf("already installed (version file exists at %s) — use 'teamwork update' instead, or --force to reinstall", vp)
	}

	files, commitSHA, err := fetchTarball(owner, repo, ref)
	if err != nil {
		return fmt.Errorf("fetching tarball: %w", err)
	}

	files = filterLanguageFiles(files, dir)

	m := &Manifest{
		Version: commitSHA,
		Files:   make(map[string]string),
	}

	written := 0
	for _, f := range files {
		dst := filepath.Join(dir, f.Path)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", f.Path, err)
		}
		if err := os.WriteFile(dst, f.Data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", f.Path, err)
		}
		m.Files[f.Path] = sha256hex(f.Data)
		written++
	}

	// Create starter files if absent.
	starterCreated := 0
	for relPath, content := range StarterTemplates {
		dst := filepath.Join(dir, relPath)
		if _, err := os.Stat(dst); err == nil {
			continue // already exists
		}
		if err := os.WriteFile(dst, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing starter %s: %w", relPath, err)
		}
		starterCreated++
	}
	// Create README.md if absent.
	readmePath := filepath.Join(dir, "README.md")
	if _, err := os.Stat(readmePath); err != nil {
		if err := os.WriteFile(readmePath, []byte("# Project\n"), 0o644); err != nil {
			return fmt.Errorf("writing README.md: %w", err)
		}
		starterCreated++
	}

	// Initialize .teamwork/ subdirectories.
	teamworkDir := filepath.Join(dir, ".teamwork")
	subdirs := []string{"state", "handoffs", "memory", "metrics"}
	for _, sub := range subdirs {
		if err := os.MkdirAll(filepath.Join(teamworkDir, sub), 0o755); err != nil {
			return fmt.Errorf("creating .teamwork/%s: %w", sub, err)
		}
	}

	if err := writeManifest(dir, m); err != nil {
		return err
	}
	if err := writeVersion(dir, commitSHA); err != nil {
		return err
	}

	fmt.Printf("Installed %d framework files, created %d starter files (version %s)\n", written, starterCreated, shortSHA(commitSHA))
	return nil
}

// Update fetches the latest framework files and updates changed files.
// User-modified files are skipped unless force is true.
func Update(dir, owner, repo, ref string, force bool) error {
	currentVersion, err := readVersion(dir)
	if err != nil {
		fmt.Println("Warning: no version file found — treating as untracked install.")
	}

	oldManifest, err := readManifest(dir)
	if err != nil {
		oldManifest = &Manifest{Files: make(map[string]string)}
	}

	files, commitSHA, err := fetchTarball(owner, repo, ref)
	if err != nil {
		return fmt.Errorf("fetching tarball: %w", err)
	}

	files = filterLanguageFiles(files, dir)

	if currentVersion == commitSHA {
		fmt.Println("Already up to date.")
		return nil
	}

	newManifest := &Manifest{
		Version: commitSHA,
		Files:   make(map[string]string),
	}

	updated, skipped, upToDate := 0, 0, 0
	var skippedPaths []string

	for _, f := range files {
		newHash := sha256hex(f.Data)
		newManifest.Files[f.Path] = newHash
		dst := filepath.Join(dir, f.Path)

		existing, err := os.ReadFile(dst)
		if err != nil {
			// File doesn't exist locally — write it.
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return fmt.Errorf("creating directory for %s: %w", f.Path, err)
			}
			if err := os.WriteFile(dst, f.Data, 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", f.Path, err)
			}
			updated++
			continue
		}

		currentHash := sha256hex(existing)
		manifestHash := oldManifest.Files[f.Path]

		if currentHash == newHash {
			// Already matches new version.
			upToDate++
			continue
		}

		if currentHash == manifestHash || manifestHash == "" {
			// User hasn't modified it (or untracked) — overwrite.
			if err := os.WriteFile(dst, f.Data, 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", f.Path, err)
			}
			updated++
			continue
		}

		// User modified the file.
		if force {
			if err := os.WriteFile(dst, f.Data, 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", f.Path, err)
			}
			updated++
			continue
		}

		skipped++
		skippedPaths = append(skippedPaths, f.Path)
	}

	if err := writeManifest(dir, newManifest); err != nil {
		return err
	}
	if err := writeVersion(dir, commitSHA); err != nil {
		return err
	}

	// Ensure .teamwork/ subdirectories exist (parity with Install).
	teamworkDir := filepath.Join(dir, ".teamwork")
	subdirs := []string{"state", "handoffs", "memory", "metrics"}
	for _, sub := range subdirs {
		if err := os.MkdirAll(filepath.Join(teamworkDir, sub), 0o755); err != nil {
			return fmt.Errorf("creating .teamwork/%s: %w", sub, err)
		}
	}

	// Clean up deprecated files from previous versions.
	removed := cleanDeprecatedFiles(dir, oldManifest)

	// If deprecated files were cleaned up, append a migration note to MEMORY.md.
	if removed > 0 {
		appendMigrationNote(dir)
	}

	for _, p := range skippedPaths {
		fmt.Printf("  skipped (user-modified): %s\n", p)
	}
	fmt.Printf("Updated %d, skipped %d (user-modified), %d already up to date, %d deprecated removed (version %s)\n",
		updated, skipped, upToDate, removed, shortSHA(commitSHA))

	// Check for unfilled CUSTOMIZE placeholders in agent files and remind the user.
	if placeholders := CustomizePlaceholderFiles(dir); len(placeholders) > 0 {
		fmt.Printf("\n  %d agent file(s) have unfilled <!-- CUSTOMIZE --> placeholders.\n", len(placeholders))
		fmt.Println("  Run the /setup-teamwork skill in GitHub Copilot to auto-detect your tech stack and fill them in.")
	}

	return nil
}

// fetchTarball downloads the repo tarball and extracts framework files.
// Returns the extracted files, the commit SHA, and any error.
func fetchTarball(owner, repo, ref string) ([]File, string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/tarball/%s", owner, repo, ref)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build request: %w", err)
	}
	// Use GH_TOKEN or GITHUB_TOKEN if available (required for private repos).
	if token := githubToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("HTTP GET %s: %w", apiURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, apiURL)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var files []File
	var prefix string
	commitSHA := ""

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("tar: %w", err)
		}

		// Skip PAX global/extended headers — they precede the real entries.
		if hdr.Typeflag == tar.TypeXGlobalHeader || hdr.Typeflag == tar.TypeXHeader {
			continue
		}

		// Determine the top-level prefix from the first entry.
		if prefix == "" {
			parts := strings.SplitN(hdr.Name, "/", 2)
			if len(parts) > 0 {
				prefix = parts[0] + "/"
				// Extract commit SHA from prefix (e.g., "JoshLuedeman-teamwork-abc1234/")
				idx := strings.LastIndex(parts[0], "-")
				if idx >= 0 {
					commitSHA = parts[0][idx+1:]
				}
			}
		}

		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		// Strip prefix to get relative path.
		relPath := strings.TrimPrefix(hdr.Name, prefix)
		if relPath == "" {
			continue
		}

		// Prevent path traversal (zip slip).
		relPath = pathpkg.Clean(relPath)
		if relPath == "." || strings.HasPrefix(relPath, "..") || pathpkg.IsAbs(relPath) {
			continue
		}

		if !isFrameworkFile(relPath) {
			continue
		}

		const maxFileSize = 10 * 1024 * 1024 // 10MB
		data, err := io.ReadAll(io.LimitReader(tr, maxFileSize+1))
		if err != nil {
			return nil, "", fmt.Errorf("reading %s: %w", relPath, err)
		}
		if int64(len(data)) > maxFileSize {
			return nil, "", fmt.Errorf("file %s exceeds maximum size of 10MB", relPath)
		}

		files = append(files, File{Path: relPath, Data: data})
	}

	if commitSHA == "" {
		return nil, "", fmt.Errorf("could not determine commit SHA from tarball")
	}

	return files, commitSHA, nil
}

func isFrameworkFile(path string) bool {
	for _, prefix := range FrameworkFiles {
		if strings.HasSuffix(prefix, "/") {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		} else {
			if path == prefix {
				return true
			}
		}
	}
	return false
}

// deprecatedFiles lists files and directories from previous Teamwork versions
// that should be removed on update. Entries ending in "/" are directories.
var deprecatedFiles = []string{
	"agents/roles/",
	"agents/workflows/",
	"CLAUDE.md",
	".cursorrules",
}

// deprecatedFileMapping maps individual old file paths to their new locations.
// Used to migrate user modifications from deprecated files into new files.
var deprecatedFileMapping = map[string]string{
	// Roles → Custom Agents
	"agents/roles/planner.md":                    ".github/agents/planner.agent.md",
	"agents/roles/architect.md":                  ".github/agents/architect.agent.md",
	"agents/roles/coder.md":                      ".github/agents/coder.agent.md",
	"agents/roles/tester.md":                     ".github/agents/tester.agent.md",
	"agents/roles/reviewer.md":                   ".github/agents/reviewer.agent.md",
	"agents/roles/security-auditor.md":           ".github/agents/security-auditor.agent.md",
	"agents/roles/documenter.md":                 ".github/agents/documenter.agent.md",
	"agents/roles/orchestrator.md":               ".github/agents/orchestrator.agent.md",
	"agents/roles/optional/triager.md":           ".github/agents/triager.agent.md",
	"agents/roles/optional/devops.md":            ".github/agents/devops.agent.md",
	"agents/roles/optional/dependency-manager.md": ".github/agents/dependency-manager.agent.md",
	"agents/roles/optional/refactorer.md":        ".github/agents/refactorer.agent.md",
	// Workflows → Skills
	"agents/workflows/feature.md":           ".github/skills/feature-workflow/SKILL.md",
	"agents/workflows/bugfix.md":            ".github/skills/bugfix-workflow/SKILL.md",
	"agents/workflows/refactor.md":          ".github/skills/refactor-workflow/SKILL.md",
	"agents/workflows/hotfix.md":            ".github/skills/hotfix-workflow/SKILL.md",
	"agents/workflows/security-response.md": ".github/skills/security-response/SKILL.md",
	"agents/workflows/dependency-update.md": ".github/skills/dependency-update/SKILL.md",
	"agents/workflows/documentation.md":     ".github/skills/documentation-workflow/SKILL.md",
	"agents/workflows/spike.md":             ".github/skills/spike-workflow/SKILL.md",
	"agents/workflows/release.md":           ".github/skills/release-workflow/SKILL.md",
	"agents/workflows/rollback.md":          ".github/skills/rollback-workflow/SKILL.md",
	// Single files
	"CLAUDE.md":    ".github/copilot-instructions.md",
	".cursorrules": ".github/copilot-instructions.md",
}

// cleanDeprecatedFiles removes deprecated files/directories, migrating any
// user modifications into the corresponding new files before removal.
// Returns the number of items removed.
func cleanDeprecatedFiles(dir string, oldManifest *Manifest) int {
	removed := 0
	for _, dep := range deprecatedFiles {
		target := filepath.Join(dir, dep)

		if strings.HasSuffix(dep, "/") {
			if _, err := os.Stat(target); os.IsNotExist(err) {
				continue
			}
			// Directory: migrate each modified file, then remove the directory.
			migrated := 0
			for manifestPath, manifestHash := range oldManifest.Files {
				if !strings.HasPrefix(manifestPath, dep) {
					continue
				}
				existing, err := os.ReadFile(filepath.Join(dir, manifestPath))
				if err != nil {
					continue
				}
				if sha256hex(existing) != manifestHash {
					// User modified this file — migrate content to new location.
					if newPath, ok := deprecatedFileMapping[manifestPath]; ok {
						migrateContent(dir, manifestPath, newPath, existing)
						migrated++
					}
				}
			}
			// Also check for files not in manifest (user-created files).
			filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				relPath, _ := filepath.Rel(dir, path)
				if _, inManifest := oldManifest.Files[relPath]; !inManifest {
					if newPath, ok := deprecatedFileMapping[relPath]; ok {
						data, readErr := os.ReadFile(path)
						if readErr == nil {
							migrateContent(dir, relPath, newPath, data)
							migrated++
						}
					}
				}
				return nil
			})
			if err := os.RemoveAll(target); err == nil {
				removed++
				if migrated > 0 {
					fmt.Printf("  migrated %d user-modified file(s) and removed: %s\n", migrated, dep)
				} else {
					fmt.Printf("  removed deprecated: %s\n", dep)
				}
			}
		} else {
			// Single file.
			existing, err := os.ReadFile(target)
			if err != nil {
				continue
			}
			manifestHash := oldManifest.Files[dep]
			if manifestHash != "" && sha256hex(existing) != manifestHash {
				// User modified — migrate content to new location.
				if newPath, ok := deprecatedFileMapping[dep]; ok {
					migrateContent(dir, dep, newPath, existing)
					fmt.Printf("  migrated user changes and removed: %s → %s\n", dep, newPath)
				}
			} else {
				fmt.Printf("  removed deprecated: %s\n", dep)
			}
			if err := os.Remove(target); err == nil {
				removed++
			}
		}
	}
	return removed
}

const migrateMarker = "\n\n<!-- MIGRATED FROM %s — review the content below and integrate it into this file, then delete this section -->\n\n"

// migrateContent appends user-modified content from a deprecated file to the
// corresponding new file with a clear migration marker.
func migrateContent(dir, oldPath, newPath string, content []byte) {
	dst := filepath.Join(dir, newPath)
	f, err := os.OpenFile(dst, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Printf("    warning: could not migrate %s → %s: %v\n", oldPath, newPath, err)
		return
	}
	defer f.Close()
	fmt.Fprintf(f, migrateMarker, oldPath)
	f.Write(content)
}

const migrationNote = `
- **Teamwork update — structure migration:** Roles moved from ` + "`agents/roles/`" + ` to ` + "`.github/agents/*.agent.md`" + ` (Custom Agents — selectable from Copilot dropdown). Workflows moved from ` + "`agents/workflows/`" + ` to ` + "`.github/skills/*/SKILL.md`" + ` (Skills — invocable via ` + "`/skill-name`" + `). ` + "`CLAUDE.md`" + ` and ` + "`.cursorrules`" + ` removed. Agent files with ` + "`<!-- CUSTOMIZE -->`" + ` placeholders can be configured by running ` + "`/setup-teamwork`" + ` in GitHub Copilot.
`

// appendMigrationNote adds a one-time migration note to MEMORY.md so agents
// in the updated repository know about the structural change.
func appendMigrationNote(dir string) {
	memoryPath := filepath.Join(dir, "MEMORY.md")
	data, err := os.ReadFile(memoryPath)
	if err != nil {
		return // MEMORY.md doesn't exist; nothing to append to.
	}

	content := string(data)
	// Don't append if the note is already present.
	if strings.Contains(content, "structure migration") {
		return
	}

	f, err := os.OpenFile(memoryPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprint(f, migrationNote)
	fmt.Println("  added migration note to MEMORY.md")
}

const customizePlaceholder = "<!-- CUSTOMIZE"

// CustomizePlaceholderFiles returns the names of .agent.md files under
// .github/agents/ that still contain unfilled <!-- CUSTOMIZE --> placeholders.
func CustomizePlaceholderFiles(dir string) []string {
	agentsDir := filepath.Join(dir, ".github", "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".agent.md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(agentsDir, e.Name()))
		if err != nil {
			continue
		}
		if strings.Contains(string(data), customizePlaceholder) {
			files = append(files, e.Name())
		}
	}
	return files
}

// filterLanguageFiles removes language-specific instruction files when the
// corresponding language is not detected in dir.
func filterLanguageFiles(files []File, dir string) []File {
	result := make([]File, 0, len(files))
	for _, f := range files {
		markers, ok := languageInstructionFiles[f.Path]
		if ok && !hasAnyFile(dir, markers) {
			continue
		}
		result = append(result, f)
	}
	return result
}

// hasAnyFile reports whether any of the named files exist in dir.
func hasAnyFile(dir string, names []string) bool {
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func readManifest(dir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, manifestPath))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	return &m, nil
}

func writeManifest(dir string, m *Manifest) error {
	p := filepath.Join(dir, manifestPath)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("creating manifest dir: %w", err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(p, data, 0o644)
}

func readVersion(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, versionPath))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func writeVersion(dir string, sha string) error {
	p := filepath.Join(dir, versionPath)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("creating version dir: %w", err)
	}
	return os.WriteFile(p, []byte(sha+"\n"), 0o644)
}

func sha256hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// githubToken returns the GitHub token from environment variables, or empty string.
func githubToken() string {
	if t := os.Getenv("GH_TOKEN"); t != "" {
		return t
	}
	return os.Getenv("GITHUB_TOKEN")
}

// shortSHA returns the first 12 characters of a SHA, or the full string if shorter.
func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
