// Package gates provides utilities for running custom quality gate scripts
// during the workflow handoff process.
//
// Gate scripts are shell commands configured under extra_gates in
// .teamwork/config.yaml. Each script is executed with the workflow directory
// as the working directory and must exit 0 to pass.
package gates

import (
"bytes"
"context"
"fmt"
"os/exec"
"strings"
"time"
)

// gateTimeout is the maximum time a single gate script may run.
const gateTimeout = 30 * time.Second

// GateFailure records a gate script that exited non-zero.
type GateFailure struct {
Script string
Output string
}

// RunGate executes a single gate script and reports whether it passed.
//
// The script is run via "sh -c <script>" with dir as the working directory.
// Gate passes when exit code is 0. Combined stdout+stderr is returned as
// output regardless of pass/fail.
func RunGate(dir, script string) (passed bool, output string, err error) {
ctx, cancel := context.WithTimeout(context.Background(), gateTimeout)
defer cancel()

cmd := exec.CommandContext(ctx, "sh", "-c", script)
cmd.Dir = dir

var buf bytes.Buffer
cmd.Stdout = &buf
cmd.Stderr = &buf

runErr := cmd.Run()
output = strings.TrimRight(buf.String(), "\n")

if ctx.Err() != nil {
return false, output, fmt.Errorf("gates: script %q timed out after %s", script, gateTimeout)
}

if runErr != nil {
// A non-zero exit is a gate failure, not a Go-level error.
if _, ok := runErr.(*exec.ExitError); ok {
return false, output, nil
}
return false, output, fmt.Errorf("gates: run %q: %w", script, runErr)
}

return true, output, nil
}

// RunAll executes all gate scripts in order and collects every failure.
//
// Execution continues past individual gate failures so all results are
// reported at once. A non-nil err is returned only when a script cannot be
// invoked at all (e.g. "sh" is not on PATH). Gate exit-code failures appear
// in the returned failures slice.
func RunAll(dir string, scripts []string) (failures []GateFailure, err error) {
for _, script := range scripts {
passed, output, runErr := RunGate(dir, script)
if runErr != nil {
return failures, runErr
}
if !passed {
failures = append(failures, GateFailure{Script: script, Output: output})
}
}
return failures, nil
}
