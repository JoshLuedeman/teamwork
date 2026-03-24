package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// makeTarball builds a synthetic .tar.gz with the given files nested under prefix.
func makeTarball(prefix string, files map[string]string) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Write a directory entry for the prefix so the code picks it up.
	_ = tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     prefix,
		Mode:     0o755,
	})

	for name, content := range files {
		data := []byte(content)
		_ = tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     prefix + name,
			Mode:     0o644,
			Size:     int64(len(data)),
		})
		_, _ = tw.Write(data)
	}
	tw.Close()
	gw.Close()
	return buf.Bytes()
}

// newTestServer returns an httptest.Server that serves the provided tarball bytes.
// GitHub's tarball endpoint redirects; we simulate that by sending the bytes directly.
func newTestServer(tb []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(tb)
	}))
}

// sampleFiles returns a set of framework and non-framework files for test tarballs.
func sampleFrameworkContent() map[string]string {
	return map[string]string{
		".github/agents/roles/coder.md":            "# Coder role\n",
		".github/skills/skill.md":                  "# Skill\n",
		".github/instructions/inst.md":             "# Instructions\n",
		".github/instructions/go.instructions.md":  "# Go Guidelines\n",
		".github/copilot-instructions.md":          "instructions\n",
		".github/ISSUE_TEMPLATE/bug.md":            "# Bug\n",
		".github/PULL_REQUEST_TEMPLATE.md":         "# PR template\n",
		"docs/conventions.md":                      "# Conventions\n",
		"scripts/build.sh":                         "#!/bin/bash\ngo build ./cmd/...\n",
		"scripts/test.sh":                          "#!/bin/bash\ngo test ./...\n",
		".editorconfig":                            "root = true\n",
		".pre-commit-config.yaml":                  "repos: []\n",
		".teamwork/config.yaml":                    "model_tiers:\n  premium: claude-opus\n",
		"Makefile":                                 "build:\n\t@bash scripts/build.sh\n",
		// Non-framework files — should be skipped:
		"go.mod":              "module example\n",
		"cmd/main.go":         "package main\n",
		"internal/foo/foo.go": "package foo\n",
		"Dockerfile":          "FROM scratch\n",
	}
}

// -- sha256hex --

func TestSHA256Hex_KnownContent(t *testing.T) {
	input := []byte("hello")
	h := sha256.Sum256(input)
	want := hex.EncodeToString(h[:])
	got := sha256hex(input)
	if got != want {
		t.Errorf("sha256hex = %q, want %q", got, want)
	}
}

func TestSHA256Hex_Empty(t *testing.T) {
	got := sha256hex([]byte{})
	// SHA-256 of empty string is well-known
	const want = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Errorf("sha256hex('') = %q, want %q", got, want)
	}
}

// -- isFrameworkFile --

func TestIsFrameworkFile_Included(t *testing.T) {
	cases := []string{
		".github/agents/roles/coder.md",
		".github/skills/skill.md",
		".github/instructions/inst.md",
		".github/copilot-instructions.md",
		".github/ISSUE_TEMPLATE/bug.md",
		".github/PULL_REQUEST_TEMPLATE.md",
		"docs/conventions.md",
		"docs/glossary.md",
		"scripts/build.sh",
		"scripts/lint.sh",
		".editorconfig",
		".pre-commit-config.yaml",
		".teamwork/config.yaml",
		"Makefile",
	}
	for _, c := range cases {
		if !isFrameworkFile(c) {
			t.Errorf("expected %q to be a framework file", c)
		}
	}
}

func TestIsFrameworkFile_Excluded(t *testing.T) {
	cases := []string{
		"go.mod",
		"go.sum",
		"cmd/main.go",
		"internal/installer/installer.go",
		"Dockerfile",
		"MEMORY.md",
		"CHANGELOG.md",
		"README.md",
		"CLAUDE.md",
		".cursorrules",
		"agents/roles/coder.md",
		".github/workflows/ci.yaml",
	}
	for _, c := range cases {
		if isFrameworkFile(c) {
			t.Errorf("expected %q to NOT be a framework file", c)
		}
	}
}

// -- writeVersion / readVersion --

func TestWriteReadVersion(t *testing.T) {
	dir := t.TempDir()
	sha := "abc1234def5678abc1234def5678abc1234def56"
	if err := writeVersion(dir, sha); err != nil {
		t.Fatalf("writeVersion: %v", err)
	}
	got, err := readVersion(dir)
	if err != nil {
		t.Fatalf("readVersion: %v", err)
	}
	if got != sha {
		t.Errorf("readVersion = %q, want %q", got, sha)
	}
}

func TestReadVersion_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := readVersion(dir)
	if err == nil {
		t.Error("expected error reading version from empty dir")
	}
}

// -- writeManifest / readManifest --

func TestWriteReadManifest(t *testing.T) {
	dir := t.TempDir()
	m := &Manifest{
		Version: "sha123",
		Files: map[string]string{
			".github/agents/roles/coder.md": "deadbeef",
			".editorconfig":                 "cafebabe",
		},
	}
	if err := writeManifest(dir, m); err != nil {
		t.Fatalf("writeManifest: %v", err)
	}
	got, err := readManifest(dir)
	if err != nil {
		t.Fatalf("readManifest: %v", err)
	}
	if got.Version != m.Version {
		t.Errorf("Version = %q, want %q", got.Version, m.Version)
	}
	for k, v := range m.Files {
		if got.Files[k] != v {
			t.Errorf("Files[%q] = %q, want %q", k, got.Files[k], v)
		}
	}
}

func TestReadManifest_MissingFile(t *testing.T) {
	dir := t.TempDir()
	// The implementation returns an error when manifest is missing;
	// callers treat that as an empty manifest (see Update()).
	_, err := readManifest(dir)
	if err == nil {
		t.Error("expected error reading manifest from empty dir")
	}
}

func TestReadManifest_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	content := `{"version":"v1","files":{".editorconfig":"aabbcc"}}`
	p := filepath.Join(dir, manifestPath)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte(content), 0o644)

	m, err := readManifest(dir)
	if err != nil {
		t.Fatalf("readManifest: %v", err)
	}
	if m.Version != "v1" {
		t.Errorf("Version = %q, want v1", m.Version)
	}
	if m.Files[".editorconfig"] != "aabbcc" {
		t.Errorf("Files[.editorconfig] = %q, want aabbcc", m.Files[".editorconfig"])
	}
}

func TestReadManifest_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, manifestPath)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte("not-json"), 0o644)
	_, err := readManifest(dir)
	if err == nil {
		t.Error("expected error for invalid JSON manifest")
	}
}

// -- Install (via mock HTTP server) --



// -- Lower-level tarball parsing tests --

// parseTarball is a test helper that decodes a tarball and returns (files, sha, err).
// It mirrors the logic inside fetchTarball without the HTTP layer.
func decodeTarball(data []byte) ([]File, string, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var files []File
	var prefix, commitSHA string

	for {
		hdr, err := tr.Next()
		if err != nil {
			break // io.EOF or other terminal error
		}
		if prefix == "" {
			parts := splitN(hdr.Name, "/", 2)
			if len(parts) > 0 {
				prefix = parts[0] + "/"
				idx := lastIndex(parts[0], "-")
				if idx >= 0 {
					commitSHA = parts[0][idx+1:]
				}
			}
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		relPath := trimPrefix(hdr.Name, prefix)
		if relPath == "" || !isFrameworkFile(relPath) {
			continue
		}
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(tr)
		files = append(files, File{Path: relPath, Data: buf.Bytes()})
	}

	return files, commitSHA, nil
}

// tiny helpers to avoid importing strings in a test-only context — we just call
// the standard library directly.
func splitN(s, sep string, n int) []string {
	var result []string
	for i := 0; i < n-1; i++ {
		idx := indexOf(s, sep)
		if idx < 0 {
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	return append(result, s)
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func lastIndex(s, sub string) int {
	last := -1
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			last = i
		}
	}
	return last
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

// -- Tarball parsing tests --

const testPrefix = "JoshLuedeman-teamwork-abc1234abc1234/"

func TestTarballParsing_FrameworkFilesExtracted(t *testing.T) {
	tb := makeTarball(testPrefix, sampleFrameworkContent())
	files, sha, err := decodeTarball(tb)
	if err != nil {
		t.Fatalf("decodeTarball: %v", err)
	}
	if sha != "abc1234abc1234" {
		t.Errorf("commitSHA = %q, want abc1234abc1234", sha)
	}

	paths := make(map[string]bool, len(files))
	for _, f := range files {
		paths[f.Path] = true
	}

	// Should be included:
	for _, want := range []string{
		".github/agents/roles/coder.md",
		"docs/conventions.md",
		"scripts/build.sh",
		"scripts/test.sh",
		".editorconfig",
		".pre-commit-config.yaml",
		".github/copilot-instructions.md",
		".teamwork/config.yaml",
		"Makefile",
	} {
		if !paths[want] {
			t.Errorf("expected framework file %q to be extracted", want)
		}
	}
}

func TestTarballParsing_NonFrameworkFilesSkipped(t *testing.T) {
	tb := makeTarball(testPrefix, sampleFrameworkContent())
	files, _, err := decodeTarball(tb)
	if err != nil {
		t.Fatalf("decodeTarball: %v", err)
	}

	paths := make(map[string]bool, len(files))
	for _, f := range files {
		paths[f.Path] = true
	}

	// Should be excluded:
	for _, notWant := range []string{
		"go.mod",
		"cmd/main.go",
		"internal/foo/foo.go",
		"Dockerfile",
	} {
		if paths[notWant] {
			t.Errorf("non-framework file %q should NOT have been extracted", notWant)
		}
	}
}

func TestTarballParsing_PrefixStripped(t *testing.T) {
	tb := makeTarball(testPrefix, map[string]string{
		".editorconfig": "content",
	})
	files, _, err := decodeTarball(tb)
	if err != nil {
		t.Fatalf("decodeTarball: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != ".editorconfig" {
		t.Errorf("path = %q, want .editorconfig (prefix should be stripped)", files[0].Path)
	}
}

func TestTarballParsing_FileContentPreserved(t *testing.T) {
	const content = "# Coder role\nSome content here.\n"
	tb := makeTarball(testPrefix, map[string]string{
		".editorconfig": content,
	})
	files, _, err := decodeTarball(tb)
	if err != nil {
		t.Fatalf("decodeTarball: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if string(files[0].Data) != content {
		t.Errorf("file content = %q, want %q", files[0].Data, content)
	}
}

// -- Install via real fetchTarball logic (mock HTTP) --

// serveAndInstall spins up a mock server that serves tb, then calls Install()
// after patching the default transport to redirect api.github.com to the server.
func serveAndInstall(t *testing.T, dir string, tb []byte) error {
	t.Helper()
	srv := newTestServer(tb)
	t.Cleanup(srv.Close)

	// Patch default http.Transport to rewrite github API calls to our server.
	original := http.DefaultTransport
	http.DefaultTransport = &redirectTransport{target: srv.URL, base: &http.Transport{}}
	t.Cleanup(func() { http.DefaultTransport = original })

	return Install(dir, "JoshLuedeman", "teamwork", "main")
}

// serveAndUpdate calls Update() with the same transport patching.
func serveAndUpdate(t *testing.T, dir string, tb []byte, force bool) error {
	t.Helper()
	srv := newTestServer(tb)
	t.Cleanup(srv.Close)

	original := http.DefaultTransport
	http.DefaultTransport = &redirectTransport{target: srv.URL, base: &http.Transport{}}
	t.Cleanup(func() { http.DefaultTransport = original })

	return Update(dir, "JoshLuedeman", "teamwork", "main", force)
}

// redirectTransport rewrites all requests to the given target URL.
// It uses a fresh *http.Transport so chained calls don't overwrite each other.
type redirectTransport struct {
	target string
	base   *http.Transport
}

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Replace the host/scheme with our test server.
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = "http"
	req2.URL.Host = stripScheme(rt.target)
	return rt.base.RoundTrip(req2)
}

func stripScheme(u string) string {
	for _, prefix := range []string{"https://", "http://"} {
		if len(u) > len(prefix) && u[:len(prefix)] == prefix {
			return u[len(prefix):]
		}
	}
	return u
}

// -- Install behaviour tests --

func TestInstall_CleanDir_FrameworkFilesWritten(t *testing.T) {
	dir := t.TempDir()
	tb := makeTarball(testPrefix, sampleFrameworkContent())

	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("Install: %v", err)
	}

	for _, want := range []string{
		".github/agents/roles/coder.md",
		"docs/conventions.md",
		"scripts/build.sh",
		"scripts/test.sh",
		".editorconfig",
		".pre-commit-config.yaml",
		".teamwork/config.yaml",
		"Makefile",
	} {
		if _, err := os.Stat(filepath.Join(dir, want)); err != nil {
			t.Errorf("framework file %q not written: %v", want, err)
		}
	}
}

func TestInstall_CleanDir_StarterFilesCreated(t *testing.T) {
	dir := t.TempDir()
	tb := makeTarball(testPrefix, sampleFrameworkContent())

	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("Install: %v", err)
	}

	for relPath := range StarterTemplates {
		if _, err := os.Stat(filepath.Join(dir, relPath)); err != nil {
			t.Errorf("starter file %q not created: %v", relPath, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err != nil {
		t.Errorf("README.md not created: %v", err)
	}
}

func TestInstall_CleanDir_ManifestAndVersionWritten(t *testing.T) {
	dir := t.TempDir()
	tb := makeTarball(testPrefix, sampleFrameworkContent())

	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("Install: %v", err)
	}

	v, err := readVersion(dir)
	if err != nil {
		t.Fatalf("readVersion after install: %v", err)
	}
	if v == "" {
		t.Error("version should not be empty after install")
	}

	m, err := readManifest(dir)
	if err != nil {
		t.Fatalf("readManifest after install: %v", err)
	}
	if len(m.Files) == 0 {
		t.Error("manifest should have file entries after install")
	}
}

func TestInstall_CleanDir_TeamworkDirInitialized(t *testing.T) {
	dir := t.TempDir()
	tb := makeTarball(testPrefix, sampleFrameworkContent())

	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("Install: %v", err)
	}

	for _, sub := range []string{"state", "handoffs", "memory", "metrics"} {
		if _, err := os.Stat(filepath.Join(dir, ".teamwork", sub)); err != nil {
			t.Errorf(".teamwork/%s not created: %v", sub, err)
		}
	}
}

func TestInstall_AlreadyInstalled_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	tb := makeTarball(testPrefix, sampleFrameworkContent())

	// First install.
	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("first Install: %v", err)
	}

	// Second install should fail.
	if err := serveAndInstall(t, dir, tb); err == nil {
		t.Error("expected error on second install, got nil")
	}
}

func TestInstall_ExistingStarterFilesNotOverwritten(t *testing.T) {
	dir := t.TempDir()

	// Pre-create a MEMORY.md with custom content.
	customContent := "# My custom memory\n"
	if err := os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte(customContent), 0o644); err != nil {
		t.Fatalf("pre-create MEMORY.md: %v", err)
	}

	tb := makeTarball(testPrefix, sampleFrameworkContent())
	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("Install: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "MEMORY.md"))
	if string(got) != customContent {
		t.Errorf("MEMORY.md overwritten; got %q, want %q", got, customContent)
	}
}

func TestInstall_MakefileAndScriptsInstalled(t *testing.T) {
	dir := t.TempDir()
	content := sampleFrameworkContent()
	tb := makeTarball(testPrefix, content)

	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("Install: %v", err)
	}

	for _, want := range []string{"Makefile", "scripts/build.sh", "scripts/test.sh"} {
		if _, err := os.Stat(filepath.Join(dir, want)); err != nil {
			t.Errorf("%s should be installed as a framework file: %v", want, err)
		}
	}
}

func TestInstall_TeamworkConfigInstalled(t *testing.T) {
	dir := t.TempDir()
	tb := makeTarball(testPrefix, sampleFrameworkContent())

	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("Install: %v", err)
	}

	cfgPath := filepath.Join(dir, ".teamwork", "config.yaml")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Errorf(".teamwork/config.yaml should be installed: %v", err)
	}
}

func TestFrameworkFiles_IncludesScriptsAndMakefile(t *testing.T) {
	found := map[string]bool{"scripts/": false, "Makefile": false, ".teamwork/config.yaml": false}
	for _, prefix := range FrameworkFiles {
		if _, ok := found[prefix]; ok {
			found[prefix] = true
		}
	}
	for entry, present := range found {
		if !present {
			t.Errorf("FrameworkFiles should include %q", entry)
		}
	}
}

// -- Language-specific instruction file tests --

func TestFilterLanguageFiles_KeepsGoInstructionsWhenGoModPresent(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644)

	files := []File{
		{Path: ".github/instructions/go.instructions.md", Data: []byte("# Go\n")},
		{Path: ".editorconfig", Data: []byte("root = true\n")},
	}
	got := filterLanguageFiles(files, dir)
	if len(got) != 2 {
		t.Errorf("expected 2 files, got %d", len(got))
	}
}

func TestFilterLanguageFiles_KeepsGoInstructionsWhenGoSumPresent(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.sum"), []byte("example v0.0.0\n"), 0o644)

	files := []File{
		{Path: ".github/instructions/go.instructions.md", Data: []byte("# Go\n")},
		{Path: ".editorconfig", Data: []byte("root = true\n")},
	}
	got := filterLanguageFiles(files, dir)
	if len(got) != 2 {
		t.Errorf("expected 2 files, got %d", len(got))
	}
}

func TestFilterLanguageFiles_DropsGoInstructionsWhenNoGoFiles(t *testing.T) {
	dir := t.TempDir()

	files := []File{
		{Path: ".github/instructions/go.instructions.md", Data: []byte("# Go\n")},
		{Path: ".editorconfig", Data: []byte("root = true\n")},
	}
	got := filterLanguageFiles(files, dir)
	if len(got) != 1 {
		t.Fatalf("expected 1 file, got %d", len(got))
	}
	if got[0].Path != ".editorconfig" {
		t.Errorf("expected .editorconfig, got %q", got[0].Path)
	}
}

func TestFilterLanguageFiles_KeepsNonLanguageFiles(t *testing.T) {
	dir := t.TempDir()

	files := []File{
		{Path: ".github/instructions/docs.instructions.md", Data: []byte("# Docs\n")},
		{Path: ".editorconfig", Data: []byte("root = true\n")},
	}
	got := filterLanguageFiles(files, dir)
	if len(got) != 2 {
		t.Errorf("expected 2 files (non-language-specific), got %d", len(got))
	}
}

func TestInstall_WithGoMod_GoInstructionsInstalled(t *testing.T) {
	dir := t.TempDir()
	// Pre-create go.mod to simulate a Go project.
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644)

	tb := makeTarball(testPrefix, sampleFrameworkContent())
	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("Install: %v", err)
	}

	goInstPath := filepath.Join(dir, ".github", "instructions", "go.instructions.md")
	if _, err := os.Stat(goInstPath); err != nil {
		t.Errorf("go.instructions.md should be installed in Go project: %v", err)
	}
}

func TestInstall_WithoutGoMod_GoInstructionsNotInstalled(t *testing.T) {
	dir := t.TempDir()
	// No go.mod — not a Go project.

	tb := makeTarball(testPrefix, sampleFrameworkContent())
	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("Install: %v", err)
	}

	goInstPath := filepath.Join(dir, ".github", "instructions", "go.instructions.md")
	if _, err := os.Stat(goInstPath); !os.IsNotExist(err) {
		t.Errorf("go.instructions.md should NOT be installed in non-Go project")
	}
}

func TestUpdate_WithGoMod_GoInstructionsInstalled(t *testing.T) {
	dir := t.TempDir()
	// Pre-create go.mod to simulate a Go project.
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644)

	tb1 := makeTarball(testPrefix, map[string]string{
		".editorconfig": "v1\n",
	})
	if err := serveAndInstall(t, dir, tb1); err != nil {
		t.Fatalf("Install: %v", err)
	}

	const newPrefix = "JoshLuedeman-teamwork-def5678def5678/"
	tb2 := makeTarball(newPrefix, map[string]string{
		".editorconfig":                           "v2\n",
		".github/instructions/go.instructions.md": "# Go Guidelines\n",
	})
	if err := serveAndUpdate(t, dir, tb2, false); err != nil {
		t.Fatalf("Update: %v", err)
	}

	goInstPath := filepath.Join(dir, ".github", "instructions", "go.instructions.md")
	if _, err := os.Stat(goInstPath); err != nil {
		t.Errorf("go.instructions.md should be installed in Go project during update: %v", err)
	}
}

func TestUpdate_WithoutGoMod_GoInstructionsNotInstalled(t *testing.T) {
	dir := t.TempDir()
	// No go.mod — not a Go project.

	tb1 := makeTarball(testPrefix, map[string]string{
		".editorconfig": "v1\n",
	})
	if err := serveAndInstall(t, dir, tb1); err != nil {
		t.Fatalf("Install: %v", err)
	}

	const newPrefix = "JoshLuedeman-teamwork-def5678def5678/"
	tb2 := makeTarball(newPrefix, map[string]string{
		".editorconfig":                           "v2\n",
		".github/instructions/go.instructions.md": "# Go Guidelines\n",
	})
	if err := serveAndUpdate(t, dir, tb2, false); err != nil {
		t.Fatalf("Update: %v", err)
	}

	goInstPath := filepath.Join(dir, ".github", "instructions", "go.instructions.md")
	if _, err := os.Stat(goInstPath); !os.IsNotExist(err) {
		t.Errorf("go.instructions.md should NOT be installed in non-Go project during update")
	}
}

// -- Update behaviour tests --

func TestUpdate_UnchangedFile_Overwritten(t *testing.T) {
	dir := t.TempDir()
	tb1 := makeTarball(testPrefix, map[string]string{
		".editorconfig": "version 1\n",
	})
	if err := serveAndInstall(t, dir, tb1); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// New tarball with a different prefix (different "commit SHA") and updated content.
	const newPrefix = "JoshLuedeman-teamwork-def5678def5678/"
	tb2 := makeTarball(newPrefix, map[string]string{
		".editorconfig": "version 2\n",
	})
	if err := serveAndUpdate(t, dir, tb2, false); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, ".editorconfig"))
	if string(got) != "version 2\n" {
		t.Errorf(".editorconfig = %q, want version 2", got)
	}
}

func TestUpdate_UserModifiedFile_Skipped(t *testing.T) {
	dir := t.TempDir()
	tb1 := makeTarball(testPrefix, map[string]string{
		".editorconfig": "original\n",
	})
	if err := serveAndInstall(t, dir, tb1); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// User modifies .editorconfig.
	userContent := "my custom edits\n"
	if err := os.WriteFile(filepath.Join(dir, ".editorconfig"), []byte(userContent), 0o644); err != nil {
		t.Fatalf("user modify .editorconfig: %v", err)
	}

	const newPrefix = "JoshLuedeman-teamwork-def5678def5678/"
	tb2 := makeTarball(newPrefix, map[string]string{
		".editorconfig": "upstream new\n",
	})
	if err := serveAndUpdate(t, dir, tb2, false); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// File should be preserved (skipped).
	got, _ := os.ReadFile(filepath.Join(dir, ".editorconfig"))
	if string(got) != userContent {
		t.Errorf("user-modified .editorconfig was overwritten; got %q, want %q", got, userContent)
	}
}

func TestUpdate_ForceFlag_OverwritesUserModifiedFile(t *testing.T) {
	dir := t.TempDir()
	tb1 := makeTarball(testPrefix, map[string]string{
		".editorconfig": "original\n",
	})
	if err := serveAndInstall(t, dir, tb1); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// User modifies .editorconfig.
	if err := os.WriteFile(filepath.Join(dir, ".editorconfig"), []byte("my custom edits\n"), 0o644); err != nil {
		t.Fatalf("user modify: %v", err)
	}

	const newPrefix = "JoshLuedeman-teamwork-def5678def5678/"
	upstreamContent := "upstream new\n"
	tb2 := makeTarball(newPrefix, map[string]string{
		".editorconfig": upstreamContent,
	})
	// --force should overwrite.
	if err := serveAndUpdate(t, dir, tb2, true); err != nil {
		t.Fatalf("Update --force: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, ".editorconfig"))
	if string(got) != upstreamContent {
		t.Errorf("--force update did not overwrite; got %q, want %q", got, upstreamContent)
	}
}

func TestUpdate_NewUpstreamFile_Written(t *testing.T) {
	dir := t.TempDir()
	tb1 := makeTarball(testPrefix, map[string]string{
		".editorconfig": "original\n",
	})
	if err := serveAndInstall(t, dir, tb1); err != nil {
		t.Fatalf("Install: %v", err)
	}

	const newPrefix = "JoshLuedeman-teamwork-def5678def5678/"
	tb2 := makeTarball(newPrefix, map[string]string{
		".editorconfig":       "original\n",
		"docs/new-feature.md": "brand new file\n",
	})
	if err := serveAndUpdate(t, dir, tb2, false); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "docs/new-feature.md")); err != nil {
		t.Errorf("new upstream file not written: %v", err)
	}
}

func TestUpdate_SameVersion_NoOp(t *testing.T) {
	dir := t.TempDir()
	tb := makeTarball(testPrefix, map[string]string{
		".editorconfig": "v1\n",
	})
	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Modify .editorconfig after install.
	if err := os.WriteFile(filepath.Join(dir, ".editorconfig"), []byte("modified\n"), 0o644); err != nil {
		t.Fatalf("modify: %v", err)
	}

	// Update with same tarball (same SHA "abc1234") — should be a no-op.
	if err := serveAndUpdate(t, dir, tb, false); err != nil {
		t.Fatalf("Update same version: %v", err)
	}

	// File should remain modified since Update short-circuits on same version.
	got, _ := os.ReadFile(filepath.Join(dir, ".editorconfig"))
	if string(got) != "modified\n" {
		t.Errorf("same-version update changed file; got %q", got)
	}
}

func TestUpdate_MissingManifest_TreatsAsUntracked(t *testing.T) {
	dir := t.TempDir()
	// Write a version file but no manifest.
	_ = os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755)
	_ = writeVersion(dir, "oldsha123")

	const newPrefix = "JoshLuedeman-teamwork-def5678def5678/"
	tb := makeTarball(newPrefix, map[string]string{
		".editorconfig": "fresh\n",
	})
	// Should succeed and write the file (no manifest → untracked → overwrite).
	if err := serveAndUpdate(t, dir, tb, false); err != nil {
		t.Fatalf("Update with missing manifest: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".editorconfig")); err != nil {
		t.Errorf(".editorconfig should have been written: %v", err)
	}
}

func TestUpdate_VersionAndManifestUpdated(t *testing.T) {
	dir := t.TempDir()
	tb1 := makeTarball(testPrefix, map[string]string{".editorconfig": "v1\n"})
	if err := serveAndInstall(t, dir, tb1); err != nil {
		t.Fatalf("Install: %v", err)
	}

	const newPrefix = "JoshLuedeman-teamwork-def5678def5678/"
	tb2 := makeTarball(newPrefix, map[string]string{".editorconfig": "v2\n"})
	if err := serveAndUpdate(t, dir, tb2, false); err != nil {
		t.Fatalf("Update: %v", err)
	}

	v, _ := readVersion(dir)
	if v != "def5678def5678" {
		t.Errorf("version after update = %q, want def5678def5678", v)
	}

	m, _ := readManifest(dir)
	if m.Version != "def5678def5678" {
		t.Errorf("manifest version = %q, want def5678def5678", m.Version)
	}
}

// -- CustomizePlaceholderFiles tests --

func TestCustomizePlaceholderFiles_WithPlaceholders(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".github", "agents")
	_ = os.MkdirAll(agentsDir, 0o755)

	// Agent with placeholder.
	_ = os.WriteFile(filepath.Join(agentsDir, "coder.agent.md"),
		[]byte("# Coder\n<!-- CUSTOMIZE: fill in -->\n- **Tech Stack:** [e.g., Go]\n"), 0o644)
	// Agent without placeholder.
	_ = os.WriteFile(filepath.Join(agentsDir, "tester.agent.md"),
		[]byte("# Tester\n- **Tech Stack:** Go, Python\n"), 0o644)
	// Non-agent file (should be ignored).
	_ = os.WriteFile(filepath.Join(agentsDir, "README.md"),
		[]byte("<!-- CUSTOMIZE -->\n"), 0o644)

	got := CustomizePlaceholderFiles(dir)
	if len(got) != 1 {
		t.Errorf("CustomizePlaceholderFiles returned %d files, want 1", len(got))
	}
	if len(got) == 1 && got[0] != "coder.agent.md" {
		t.Errorf("CustomizePlaceholderFiles[0] = %q, want coder.agent.md", got[0])
	}
}

func TestCustomizePlaceholderFiles_NoPlaceholders(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".github", "agents")
	_ = os.MkdirAll(agentsDir, 0o755)

	_ = os.WriteFile(filepath.Join(agentsDir, "coder.agent.md"),
		[]byte("# Coder\n- **Tech Stack:** Go\n"), 0o644)

	got := CustomizePlaceholderFiles(dir)
	if len(got) != 0 {
		t.Errorf("CustomizePlaceholderFiles returned %d files, want 0", len(got))
	}
}

func TestCustomizePlaceholderFiles_NoAgentsDir(t *testing.T) {
	dir := t.TempDir()
	got := CustomizePlaceholderFiles(dir)
	if len(got) != 0 {
		t.Errorf("CustomizePlaceholderFiles returned %d files, want 0 (no agents dir)", len(got))
	}
}

// -- Update initializes .teamwork/ subdirectories --

func TestUpdate_InitializesTeamworkSubdirs(t *testing.T) {
	dir := t.TempDir()
	// Pre-create only the version file (simulate old install without subdirs).
	_ = os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755)
	_ = writeVersion(dir, "oldsha123")

	const newPrefix = "JoshLuedeman-teamwork-def5678def5678/"
	tb := makeTarball(newPrefix, map[string]string{
		".editorconfig": "fresh\n",
	})
	if err := serveAndUpdate(t, dir, tb, false); err != nil {
		t.Fatalf("Update: %v", err)
	}

	for _, sub := range []string{"state", "handoffs", "memory", "metrics"} {
		if _, err := os.Stat(filepath.Join(dir, ".teamwork", sub)); err != nil {
			t.Errorf(".teamwork/%s not created during update: %v", sub, err)
		}
	}
}

// -- Integration test (skipped by default) --

func TestInstall_Integration(t *testing.T) {
	if os.Getenv("TEAMWORK_INTEGRATION_TESTS") == "" {
		t.Skip("set TEAMWORK_INTEGRATION_TESTS=1 to run integration tests against GitHub")
	}
	dir := t.TempDir()
	if err := Install(dir, "JoshLuedeman", "teamwork", "main"); err != nil {
		t.Fatalf("Install: %v", err)
	}
	v, err := readVersion(dir)
	if err != nil {
		t.Fatalf("readVersion: %v", err)
	}
	if len(v) < 7 {
		t.Errorf("version too short: %q", v)
	}
	// Verify at least one framework file was written.
	if _, err := os.Stat(filepath.Join(dir, ".editorconfig")); err != nil {
		t.Errorf(".editorconfig not written: %v", err)
	}

	m, err := readManifest(dir)
	if err != nil {
		t.Fatalf("readManifest: %v", err)
	}
	if _, ok := m.Files[".editorconfig"]; !ok {
		t.Error(".editorconfig not in manifest")
	}
	t.Logf("Installed version: %s (%d framework files)", v, len(m.Files))
}

// -- Manifest JSON structure test --

func TestManifest_JSONRoundTrip(t *testing.T) {
	m := &Manifest{
		Version: "sha123abc",
		Files: map[string]string{
			".editorconfig":                 sha256hex([]byte("content")),
			".github/agents/roles/coder.md": sha256hex([]byte("coder")),
		},
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Manifest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Version != m.Version {
		t.Errorf("Version = %q, want %q", got.Version, m.Version)
	}
	for k, v := range m.Files {
		if got.Files[k] != v {
			t.Errorf("Files[%q] = %q, want %q", k, got.Files[k], v)
		}
	}
}
