// Package handoff manages handoff artifacts in .teamwork/handoffs/.
// This file provides role-specific handoff templates for guided artifact creation.
package handoff

import "fmt"

// Templates maps "<workflow-type>/<from-role>-><to-role>" to a Markdown template string.
var Templates = map[string]string{
	// Feature workflow transitions
	"feature/coder->tester": `## Files Changed

<!-- List files created or modified -->

## How to Test

<!-- Step-by-step instructions for the tester -->

## Edge Cases

<!-- Known edge cases that should be verified -->

## Known Limitations

<!-- Anything intentionally out of scope or deferred -->
`,
	"feature/tester->security-auditor": `## Test Results

<!-- Summary of test run results (pass/fail counts, coverage) -->

## Coverage

<!-- Coverage percentage and any uncovered paths -->

## Areas of Concern

<!-- Code areas that may warrant closer security scrutiny -->
`,
	"feature/security-auditor->reviewer": `## Findings

<!-- List of security findings (or "None") -->

## Risk Level

<!-- Overall risk assessment: Low / Medium / High / Critical -->

## Mitigations Applied

<!-- Changes made to address findings -->
`,
	"feature/reviewer->human": `## Summary

<!-- Brief summary of what was reviewed -->

## Changes Requested

<!-- List of changes requested before approval, or "None" -->

## Approved

<!-- Approval status: yes / no / conditional -->
`,
	// Bugfix workflow transitions
	"bugfix/coder->tester": `## Files Changed

<!-- List files created or modified -->

## How to Test

<!-- Step-by-step instructions for the tester -->

## Edge Cases

<!-- Known edge cases that should be verified -->

## Known Limitations

<!-- Anything intentionally out of scope or deferred -->
`,
	"bugfix/tester->security-auditor": `## Test Results

<!-- Summary of test run results -->

## Coverage

<!-- Coverage percentage and regressions checked -->

## Areas of Concern

<!-- Code areas that may warrant closer security scrutiny -->
`,
	"bugfix/security-auditor->reviewer": `## Findings

<!-- List of security findings (or "None") -->

## Risk Level

<!-- Overall risk assessment: Low / Medium / High / Critical -->

## Mitigations Applied

<!-- Changes made to address findings -->
`,
	"bugfix/reviewer->human": `## Summary

<!-- Brief summary of what was reviewed -->

## Changes Requested

<!-- List of changes requested before approval, or "None" -->

## Approved

<!-- Approval status: yes / no / conditional -->
`,
	// Generic transitions reused across workflow types
	"refactor/coder->tester": `## Files Changed

<!-- List files created or modified -->

## How to Test

<!-- Step-by-step instructions for the tester -->

## Edge Cases

<!-- Known edge cases that should be verified -->

## Known Limitations

<!-- Anything intentionally out of scope or deferred -->
`,
	"refactor/tester->reviewer": `## Test Results

<!-- Summary of test run results -->

## Coverage

<!-- Coverage percentage -->

## Areas of Concern

<!-- Any behavioral changes observed -->
`,
	"refactor/reviewer->human": `## Summary

<!-- Brief summary of what was reviewed -->

## Changes Requested

<!-- List of changes requested before approval, or "None" -->

## Approved

<!-- Approval status: yes / no / conditional -->
`,
}

// genericTemplate is returned for transitions not in the Templates map.
const genericTemplate = `## Summary

<!-- Brief summary of work completed in this step -->

## Artifacts Produced

<!-- Files, documents, or other artifacts created -->

## Context for Next Role

<!-- What the next role needs to know to proceed -->

## Open Questions or Risks

<!-- Anything the next role should be aware of -->
`

// GenericTemplate returns the generic fallback template string so callers can
// detect when TemplateFor returns a non-specific result.
func GenericTemplate() string {
	return genericTemplate
}
// It looks up "<workflowType>/<fromRole>-><toRole>" in the Templates map.
// If no match is found it returns the generic fallback template.
func TemplateFor(workflowType, fromRole, toRole string) string {
	key := fmt.Sprintf("%s/%s->%s", workflowType, fromRole, toRole)
	if t, ok := Templates[key]; ok {
		return t
	}
	return genericTemplate
}
