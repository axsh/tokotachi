package assist

import (
	"fmt"
)

// instructionTemplate is the fixed instruction text for distill_intake_to_knowledge tasks.
const instructionTemplate = `Convert the following intake events into self-contained Knowledge Atoms.

## Rules

1. Each Knowledge Atom must be self-contained.
   Resolve references such as "this", "that", "the above approach", or "the previous design".
   Do not depend on the original chat context.

2. For each atom, specify:
   - type: one of Fact, Decision, Constraint, Pattern, Warning, Skill
   - title: a concise, self-contained title (max 200 chars)
   - body: a self-contained description (max 2000 chars)
   - importance: one of low, medium, high, critical
   - confidence: a float between 0.0 and 1.0
   - activation_hints.positive: list of situations where this knowledge is relevant
   - activation_hints.negative: list of situations where this knowledge is NOT relevant (optional)
   - source.event_ids: list of intake event IDs this atom was derived from

3. One intake event may produce zero or more Knowledge Atoms.
   Skip events that contain no meaningful long-term knowledge.

4. Output format must conform to the schema at: %s

## Submit

When done, save the result as a JSON file and run:
%s`

// buildInstruction returns the instruction text with schema path and submit command.
func buildInstruction(schemaPath, submitCmd string) string {
	return fmt.Sprintf(instructionTemplate, schemaPath, submitCmd)
}
