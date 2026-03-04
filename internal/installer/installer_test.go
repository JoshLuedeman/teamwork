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
		"agents/roles/coder.md": "# Coder role\n",
		"docs/conventions.md":   "# Conventions\n",
		"CLAUDE.md":             "# Claude instructions\n",
		".cursorrules":          "rules here\n",
		"Makefile":              "all:\n\techo ok\n",
		".github/copilot-instructions.md": "instructions\n",
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
		"agents/roles/coder.md",
		"agents/workflows/feature.md",
		"docs/conventions.md",
		"docs/glossary.md",
		"CLAUDE.md",
		".cursorrules",
		"Makefile",
		".github/copilot-instructions.md",
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
		".github/workflows/ci.yaml",
		"docs/cli.md",
		"docs/state-machines.md",
		"docs/onboarding.md",
		"docs/decisions/001-role-based-agent-framework.md",
		"docs/decisions/README.md",
	}
	for _, c := range cases {
		if isFrameworkFile(c) {
			t.Errorf("expected %q to NOT be a framework file", c)
		}
	}
}

// -- remapPath --

func TestRemapPath_AgentsRemapped(t *testing.T) {
	cases := map[string]string{
		"agents/roles/coder.md":        ".teamwork/agents/roles/coder.md",
		"agents/workflows/feature.md":  ".teamwork/agents/workflows/feature.md",
		"agents/README.md":             ".teamwork/agents/README.md",
		"docs/conventions.md":          ".teamwork/docs/conventions.md",
		"docs/glossary.md":             ".teamwork/docs/glossary.md",
	}
	for input, want := range cases {
		got := remapPath(input)
		if got != want {
			t.Errorf("remapPath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRemapPath_RootFilesUnchanged(t *testing.T) {
	cases := []string{
		"CLAUDE.md",
		".cursorrules",
		"Makefile",
		".github/copilot-instructions.md",
	}
	for _, input := range cases {
		got := remapPath(input)
		if got != input {
			t.Errorf("remapPath(%q) = %q, want unchanged", input, got)
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
			".teamwork/agents/roles/coder.md": "deadbeef",
			"CLAUDE.md":                       "cafebabe",
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
	content := `{"version":"v1","files":{"CLAUDE.md":"aabbcc"}}`
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
	if m.Files["CLAUDE.md"] != "aabbcc" {
		t.Errorf("Files[CLAUDE.md] = %q, want aabbcc", m.Files["CLAUDE.md"])
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
		// Skip PAX global/extended headers.
		if hdr.Typeflag == tar.TypeXGlobalHeader || hdr.Typeflag == tar.TypeXHeader {
			continue
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
		"agents/roles/coder.md",
		"docs/conventions.md",
		"CLAUDE.md",
		".cursorrules",
		"Makefile",
		".github/copilot-instructions.md",
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
		"CLAUDE.md": "content",
	})
	files, _, err := decodeTarball(tb)
	if err != nil {
		t.Fatalf("decodeTarball: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "CLAUDE.md" {
		t.Errorf("path = %q, want CLAUDE.md (prefix should be stripped)", files[0].Path)
	}
}

func TestTarballParsing_FileContentPreserved(t *testing.T) {
	const content = "# Coder role\nSome content here.\n"
	tb := makeTarball(testPrefix, map[string]string{
		"CLAUDE.md": content,
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

// makeTarballWithPAXHeaders builds a tarball with a leading pax_global_header,
// which GitHub includes in tarballs. This verifies the parser skips PAX entries.
func makeTarballWithPAXHeaders(prefix string, files map[string]string) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Write PAX global header (as GitHub does).
	_ = tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeXGlobalHeader,
		Name:     "pax_global_header",
		Size:     0,
	})

	// Directory entry for the real prefix.
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

func TestTarballParsing_PAXGlobalHeader(t *testing.T) {
	tb := makeTarballWithPAXHeaders(testPrefix, sampleFrameworkContent())
	files, sha, err := decodeTarball(tb)
	if err != nil {
		t.Fatalf("decodeTarball: %v", err)
	}
	if sha != "abc1234abc1234" {
		t.Errorf("commitSHA = %q, want abc1234abc1234", sha)
	}
	if len(files) == 0 {
		t.Fatal("expected at least one framework file")
	}
	paths := make(map[string]bool, len(files))
	for _, f := range files {
		paths[f.Path] = true
	}
	if !paths["agents/roles/coder.md"] {
		t.Error("expected agents/roles/coder.md to be extracted")
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
		".teamwork/agents/roles/coder.md",
		".teamwork/docs/conventions.md",
		"CLAUDE.md",
		".cursorrules",
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

// -- Update behaviour tests --

func TestUpdate_UnchangedFile_Overwritten(t *testing.T) {
	dir := t.TempDir()
	tb1 := makeTarball(testPrefix, map[string]string{
		"CLAUDE.md": "version 1\n",
	})
	if err := serveAndInstall(t, dir, tb1); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// New tarball with a different prefix (different "commit SHA") and updated content.
	const newPrefix = "JoshLuedeman-teamwork-def5678def5678/"
	tb2 := makeTarball(newPrefix, map[string]string{
		"CLAUDE.md": "version 2\n",
	})
	if err := serveAndUpdate(t, dir, tb2, false); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if string(got) != "version 2\n" {
		t.Errorf("CLAUDE.md = %q, want version 2", got)
	}
}

func TestUpdate_UserModifiedFile_Skipped(t *testing.T) {
	dir := t.TempDir()
	tb1 := makeTarball(testPrefix, map[string]string{
		"CLAUDE.md": "original\n",
	})
	if err := serveAndInstall(t, dir, tb1); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// User modifies CLAUDE.md.
	userContent := "my custom edits\n"
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(userContent), 0o644); err != nil {
		t.Fatalf("user modify CLAUDE.md: %v", err)
	}

	const newPrefix = "JoshLuedeman-teamwork-def5678def5678/"
	tb2 := makeTarball(newPrefix, map[string]string{
		"CLAUDE.md": "upstream new\n",
	})
	if err := serveAndUpdate(t, dir, tb2, false); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// File should be preserved (skipped).
	got, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if string(got) != userContent {
		t.Errorf("user-modified CLAUDE.md was overwritten; got %q, want %q", got, userContent)
	}
}

func TestUpdate_ForceFlag_OverwritesUserModifiedFile(t *testing.T) {
	dir := t.TempDir()
	tb1 := makeTarball(testPrefix, map[string]string{
		"CLAUDE.md": "original\n",
	})
	if err := serveAndInstall(t, dir, tb1); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// User modifies CLAUDE.md.
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("my custom edits\n"), 0o644); err != nil {
		t.Fatalf("user modify: %v", err)
	}

	const newPrefix = "JoshLuedeman-teamwork-def5678def5678/"
	upstreamContent := "upstream new\n"
	tb2 := makeTarball(newPrefix, map[string]string{
		"CLAUDE.md": upstreamContent,
	})
	// --force should overwrite.
	if err := serveAndUpdate(t, dir, tb2, true); err != nil {
		t.Fatalf("Update --force: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if string(got) != upstreamContent {
		t.Errorf("--force update did not overwrite; got %q, want %q", got, upstreamContent)
	}
}

func TestUpdate_NewUpstreamFile_Written(t *testing.T) {
	dir := t.TempDir()
	tb1 := makeTarball(testPrefix, map[string]string{
		"CLAUDE.md": "original\n",
	})
	if err := serveAndInstall(t, dir, tb1); err != nil {
		t.Fatalf("Install: %v", err)
	}

	const newPrefix = "JoshLuedeman-teamwork-def5678def5678/"
	tb2 := makeTarball(newPrefix, map[string]string{
		"CLAUDE.md":            "original\n",
		"agents/roles/new.md":  "brand new file\n",
	})
	if err := serveAndUpdate(t, dir, tb2, false); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".teamwork/agents/roles/new.md")); err != nil {
		t.Errorf("new upstream file not written: %v", err)
	}
}

func TestUpdate_SameVersion_NoOp(t *testing.T) {
	dir := t.TempDir()
	tb := makeTarball(testPrefix, map[string]string{
		"CLAUDE.md": "v1\n",
	})
	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Modify CLAUDE.md after install.
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("modified\n"), 0o644); err != nil {
		t.Fatalf("modify: %v", err)
	}

	// Update with same tarball (same SHA "abc1234") — should be a no-op.
	if err := serveAndUpdate(t, dir, tb, false); err != nil {
		t.Fatalf("Update same version: %v", err)
	}

	// File should remain modified since Update short-circuits on same version.
	got, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
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
		"CLAUDE.md": "fresh\n",
	})
	// Should succeed and write the file (no manifest → untracked → overwrite).
	if err := serveAndUpdate(t, dir, tb, false); err != nil {
		t.Fatalf("Update with missing manifest: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err != nil {
		t.Errorf("CLAUDE.md should have been written: %v", err)
	}
}

func TestUpdate_VersionAndManifestUpdated(t *testing.T) {
	dir := t.TempDir()
	tb1 := makeTarball(testPrefix, map[string]string{"CLAUDE.md": "v1\n"})
	if err := serveAndInstall(t, dir, tb1); err != nil {
		t.Fatalf("Install: %v", err)
	}

	const newPrefix = "JoshLuedeman-teamwork-def5678def5678/"
	tb2 := makeTarball(newPrefix, map[string]string{"CLAUDE.md": "v2\n"})
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
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err != nil {
		t.Errorf("CLAUDE.md not written: %v", err)
	}

	m, err := readManifest(dir)
	if err != nil {
		t.Fatalf("readManifest: %v", err)
	}
	if _, ok := m.Files["CLAUDE.md"]; !ok {
		t.Error("CLAUDE.md not in manifest")
	}
	t.Logf("Installed version: %s (%d framework files)", v, len(m.Files))
}

// -- Manifest JSON structure test --

func TestManifest_JSONRoundTrip(t *testing.T) {
	m := &Manifest{
		Version: "sha123abc",
		Files: map[string]string{
			"CLAUDE.md":                        sha256hex([]byte("content")),
			".teamwork/agents/roles/coder.md":  sha256hex([]byte("coder")),
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
