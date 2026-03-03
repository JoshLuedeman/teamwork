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
	"path/filepath"
	"strings"
)

// FrameworkFiles is the list of path prefixes extracted from the tarball.
// These are overwritten on update (if unchanged by user).
var FrameworkFiles = []string{
	"agents/",
	"docs/",
	".github/copilot-instructions.md",
	"CLAUDE.md",
	".cursorrules",
	"Makefile",
}

// StarterTemplates maps relative path to content for files created once on install.
// Never overwritten by update.
var StarterTemplates = map[string]string{
	"MEMORY.md": "# Project Memory\n\nThis file captures project learnings that persist across agent sessions.\n",
	"CHANGELOG.md": "# Changelog\n\nAll notable changes to this project will be documented in this file.\n\n" +
		"The format is based on [Keep a Changelog](https://keepachangelog.com/).\n",
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

	// Initialize .teamwork/ if it doesn't exist (mirrors init command logic).
	teamworkDir := filepath.Join(dir, ".teamwork")
	if _, err := os.Stat(teamworkDir); err != nil {
		subdirs := []string{"state", "handoffs", "memory", "metrics"}
		for _, sub := range subdirs {
			if err := os.MkdirAll(filepath.Join(teamworkDir, sub), 0o755); err != nil {
				return fmt.Errorf("creating .teamwork/%s: %w", sub, err)
			}
		}
	}

	if err := writeManifest(dir, m); err != nil {
		return err
	}
	if err := writeVersion(dir, commitSHA); err != nil {
		return err
	}

	fmt.Printf("Installed %d framework files, created %d starter files (version %s)\n", written, starterCreated, commitSHA[:12])
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

	for _, p := range skippedPaths {
		fmt.Printf("  skipped (user-modified): %s\n", p)
	}
	fmt.Printf("Updated %d, skipped %d (user-modified), %d already up to date (version %s)\n",
		updated, skipped, upToDate, commitSHA[:12])
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

	resp, err := http.DefaultClient.Do(req)
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

		if !isFrameworkFile(relPath) {
			continue
		}

		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, "", fmt.Errorf("reading %s: %w", relPath, err)
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
