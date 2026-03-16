package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestBuildSetupIssueBody_ContainsFileList(t *testing.T) {
	files := []string{"coder.agent.md", "tester.agent.md"}
	body := buildSetupIssueBody(files)

	for _, f := range files {
		want := "`.github/agents/" + f + "`"
		if !strings.Contains(body, want) {
			t.Errorf("body missing file reference %q", want)
		}
	}
}

func TestBuildSetupIssueBody_ContainsInstructions(t *testing.T) {
	body := buildSetupIssueBody([]string{"coder.agent.md"})

	for _, want := range []string{
		"/setup-teamwork",
		"<!-- CUSTOMIZE -->",
		"auto-detect",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing expected text %q", want)
		}
	}
}

func TestBuildSetupIssueBody_SingleFile(t *testing.T) {
	body := buildSetupIssueBody([]string{"architect.agent.md"})
	if !strings.Contains(body, "`.github/agents/architect.agent.md`") {
		t.Error("body missing architect.agent.md reference")
	}
}

func TestParseUpdateSource_Valid(t *testing.T) {
	owner, repo, err := parseUpdateSource("joshluedeman/teamwork")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "joshluedeman" || repo != "teamwork" {
		t.Errorf("got owner=%q repo=%q, want joshluedeman/teamwork", owner, repo)
	}
}

func TestParseUpdateSource_Invalid(t *testing.T) {
	cases := []string{"", "noslash", "/empty", "empty/", "/"}
	for _, c := range cases {
		_, _, err := parseUpdateSource(c)
		if err == nil {
			t.Errorf("parseUpdateSource(%q) should have returned error", c)
		}
	}
}

// ---- --check flag tests ----

const checkTestPrefix = "JoshLuedeman-teamwork-abc1234abc1234/"

// cmdTarball builds a minimal .tar.gz with files nested under prefix.
func cmdTarball(prefix string, files map[string]string) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	_ = tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: prefix, Mode: 0o755})
	for name, content := range files {
		data := []byte(content)
		_ = tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: prefix + name, Mode: 0o644, Size: int64(len(data))})
		_, _ = tw.Write(data)
	}
	tw.Close()
	gw.Close()
	return buf.Bytes()
}

// cmdTestServer returns a test server that serves the provided tarball bytes.
func cmdTestServer(tb []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(tb)
	}))
}

// cmdRedirectTransport rewrites all HTTP requests to the given target URL.
type cmdRedirectTransport struct {
	target string
	base   *http.Transport
}

func (rt *cmdRedirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = "http"
	target := rt.target
	for _, pfx := range []string{"https://", "http://"} {
		if strings.HasPrefix(target, pfx) {
			target = target[len(pfx):]
			break
		}
	}
	req2.URL.Host = target
	return rt.base.RoundTrip(req2)
}

// resetUpdateFlags resets the update command flags to defaults between tests.
func resetUpdateFlags(t *testing.T) {
	t.Helper()
	updateCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// executeUpdateCmd runs "teamwork update" with args and captures stdout/stderr.
func executeUpdateCmd(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	resetUpdateFlags(t)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"update", "--dir", dir}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

// writeTeamworkMeta writes minimal .teamwork files so CheckDrift can read state.
func writeTeamworkMeta(t *testing.T, dir string) {
	t.Helper()
	teamworkDir := filepath.Join(dir, ".teamwork")
	if err := os.MkdirAll(teamworkDir, 0o755); err != nil {
		t.Fatalf("MkdirAll .teamwork: %v", err)
	}
	_ = os.WriteFile(filepath.Join(teamworkDir, "framework-version.txt"), []byte("abc1234abc1234\n"), 0o644)
	_ = os.WriteFile(filepath.Join(teamworkDir, "framework-manifest.json"),
		[]byte(`{"version":"abc1234abc1234","files":{}}`), 0o644)
}

// patchHTTP redirects all HTTP requests to srv.
func patchHTTP(t *testing.T, srv *httptest.Server) {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = &cmdRedirectTransport{target: srv.URL, base: &http.Transport{}}
	t.Cleanup(func() { http.DefaultTransport = original })
}

// TestCheckFlag_NoDrift verifies --check exits 0 when local files match upstream.
func TestCheckFlag_NoDrift(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		".editorconfig":                   "root = true\n",
		".github/copilot-instructions.md": "instructions\n",
	}
	for relPath, content := range files {
		dst := filepath.Join(dir, relPath)
		_ = os.MkdirAll(filepath.Dir(dst), 0o755)
		_ = os.WriteFile(dst, []byte(content), 0o644)
	}
	writeTeamworkMeta(t, dir)
	srv := cmdTestServer(cmdTarball(checkTestPrefix, files))
	t.Cleanup(srv.Close)
	patchHTTP(t, srv)
	out, err := executeUpdateCmd(t, dir, "--check", "--source", "JoshLuedeman/teamwork")
	if err != nil {
		t.Fatalf("--check with no drift should exit 0, got: %v", err)
	}
	if !strings.Contains(out, "No drift") {
		t.Errorf("expected 'No drift' in output, got: %q", out)
	}
}

// TestCheckFlag_DriftDetected verifies --check exits 1 when local files differ.
func TestCheckFlag_DriftDetected(t *testing.T) {
	dir := t.TempDir()
	upstreamFiles := map[string]string{".editorconfig": "root = true\n"}
	_ = os.WriteFile(filepath.Join(dir, ".editorconfig"), []byte("# modified\n"), 0o644)
	writeTeamworkMeta(t, dir)
	srv := cmdTestServer(cmdTarball(checkTestPrefix, upstreamFiles))
	t.Cleanup(srv.Close)
	patchHTTP(t, srv)
	out, err := executeUpdateCmd(t, dir, "--check", "--source", "JoshLuedeman/teamwork")
	exitErr, ok := err.(*ExitError)
	if !ok || exitErr.Code != 1 {
		t.Errorf("expected ExitError{Code:1}, got: %v (type %T)", err, err)
	}
	if !strings.Contains(out, "Drift detected") {
		t.Errorf("expected 'Drift detected' in output, got: %q", out)
	}
	if !strings.Contains(out, ".editorconfig") {
		t.Errorf("expected .editorconfig in output, got: %q", out)
	}
}

// TestCheckFlag_NoFilesModified verifies --check does not write to disk.
func TestCheckFlag_NoFilesModified(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, ".editorconfig"), []byte("local\n"), 0o644)
	writeTeamworkMeta(t, dir)
	upstreamFiles := map[string]string{".editorconfig": "upstream\n"}
	srv := cmdTestServer(cmdTarball(checkTestPrefix, upstreamFiles))
	t.Cleanup(srv.Close)
	patchHTTP(t, srv)
	_, _ = executeUpdateCmd(t, dir, "--check", "--source", "JoshLuedeman/teamwork")
	data, err := os.ReadFile(filepath.Join(dir, ".editorconfig"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "local\n" {
		t.Errorf("--check modified .editorconfig: got %q, want local", data)
	}
}
