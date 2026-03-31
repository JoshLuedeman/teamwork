package gates

import (
	"fmt"
	"log"
	"os/exec"
)

// secretsScanner describes a secrets scanning tool and the command to invoke it.
type secretsScanner struct {
	binary string
	args   string
}

// scanners is the ordered list of supported secrets scanning tools.
// The first tool found in PATH is used.
var scanners = []secretsScanner{
	{"gitleaks", "detect --source . --no-git"},
	{"detect-secrets", "scan"},
	{"trufflehog", "filesystem ."},
}

// RunSecretsGate attempts to run a secrets scanner on dir using runner.
//
// It tries each supported tool in order (gitleaks, detect-secrets, trufflehog)
// and uses the first one found in PATH. If no scanner is installed, it logs a
// warning and returns found=false without error.
//
// A non-zero exit code from the scanner is interpreted as "secrets found";
// the scanner's output is returned as details.
func RunSecretsGate(dir string, runner Runner) (found bool, details string, err error) {
	for _, s := range scanners {
		if _, lookErr := exec.LookPath(s.binary); lookErr != nil {
			continue
		}
		// Found a scanner — run it.
		cmd := s.binary + " " + s.args
		out, runErr := runner.Run(cmd, dir)
		if runErr != nil {
			if _, ok := runErr.(*exec.ExitError); ok {
				// Non-zero exit → secrets found.
				return true, string(out), nil
			}
			return false, "", fmt.Errorf("gates: secrets scan (%s): %w", s.binary, runErr)
		}
		return false, string(out), nil
	}

	// No scanner found in PATH.
	log.Printf("gates: secrets scan: no scanner found in PATH (gitleaks, detect-secrets, trufflehog); skipping")
	return false, "", nil
}
