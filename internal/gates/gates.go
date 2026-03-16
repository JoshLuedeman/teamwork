package gates

import (
"fmt"
"os/exec"
)

type Runner interface {
Run(command, dir string) ([]byte, error)
}

type ShellRunner struct{}

func (ShellRunner) Run(command, dir string) ([]byte, error) {
cmd := exec.Command("/bin/sh", "-c", command)
cmd.Dir = dir
return cmd.CombinedOutput()
}

type GateResult struct {
Condition string
Output    string
Passed    bool
}

func GateKey(step int) string {
return fmt.Sprintf("after_step_%d", step)
}

func Lookup(extraGates map[string]map[string][]string, workflowType, location string) []string {
if extraGates == nil {
return nil
}
locs, ok := extraGates[workflowType]
if !ok {
return nil
}
return locs[location]
}

func RunGate(location string, conditions []string, dir string, runner Runner) ([]GateResult, bool, error) {
var results []GateResult
for _, cond := range conditions {
out, err := runner.Run(cond, dir)
if err != nil {
if _, ok := err.(*exec.ExitError); ok {
results = append(results, GateResult{Condition: cond, Output: string(out), Passed: false})
return results, false, nil
}
return results, false, fmt.Errorf("gates: run %q: %w", cond, err)
}
results = append(results, GateResult{Condition: cond, Output: string(out), Passed: true})
}
return results, true, nil
}
