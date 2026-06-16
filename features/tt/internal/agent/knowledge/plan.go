package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
)

// ParseSplitPlan reads a split operation plan from a JSON file.
func ParseSplitPlan(planFile string) (*SplitPlan, error) {
	data, err := os.ReadFile(planFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read split plan: %w", err)
	}
	var plan SplitPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse split plan: %w", err)
	}
	if len(plan.Assignments) == 0 {
		return nil, fmt.Errorf("split plan has no assignments")
	}
	return &plan, nil
}

// ParseMergePlan reads a merge operation plan from a JSON file.
func ParseMergePlan(planFile string) (*MergePlan, error) {
	data, err := os.ReadFile(planFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read merge plan: %w", err)
	}
	var plan MergePlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse merge plan: %w", err)
	}
	if plan.Title == "" {
		return nil, fmt.Errorf("merge plan requires a title")
	}
	return &plan, nil
}
