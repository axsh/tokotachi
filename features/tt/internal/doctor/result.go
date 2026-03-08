package doctor

import (
	"encoding/json"
	"fmt"
	"io"
)

// Status represents the result of a health check.
type Status int

const (
	StatusPass Status = iota
	StatusFail
	StatusWarn
)

// String returns the emoji icon for the status.
func (s Status) String() string {
	switch s {
	case StatusPass:
		return "✅"
	case StatusFail:
		return "❌"
	case StatusWarn:
		return "⚠️"
	default:
		return "?"
	}
}

// MarshalJSON serializes Status as a lowercase string.
func (s Status) MarshalJSON() ([]byte, error) {
	var label string
	switch s {
	case StatusPass:
		label = "pass"
	case StatusFail:
		label = "fail"
	case StatusWarn:
		label = "warn"
	default:
		label = "unknown"
	}
	return json.Marshal(label)
}

// CheckResult holds the outcome of a single health check.
type CheckResult struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Status   Status `json:"status"`
	Message  string `json:"message"`
	Expected string `json:"expected,omitempty"`
	FixHint  string `json:"fix_hint,omitempty"`
	Fixed    bool   `json:"fixed,omitempty"`
}

// Summary holds aggregated check counts.
type Summary struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	Warnings int `json:"warnings"`
	Fixed    int `json:"fixed,omitempty"`
}

// Report aggregates all check results.
type Report struct {
	Results []CheckResult `json:"results"`
}

// HasFailures returns true if any result has StatusFail.
func (r *Report) HasFailures() bool {
	for _, res := range r.Results {
		if res.Status == StatusFail {
			return true
		}
	}
	return false
}

// Summary returns aggregated counts of check results.
func (r *Report) Summary() Summary {
	var s Summary
	s.Total = len(r.Results)
	for _, res := range r.Results {
		switch res.Status {
		case StatusPass:
			s.Passed++
		case StatusFail:
			s.Failed++
		case StatusWarn:
			s.Warnings++
		}
		if res.Fixed {
			s.Fixed++
		}
	}
	return s
}

// PrintText writes human-readable output grouped by category.
func (r *Report) PrintText(w io.Writer) {
	fmt.Fprintln(w, "🏥 tt doctor")
	fmt.Fprintln(w, "================")
	fmt.Fprintln(w)

	// Group results by category preserving order.
	type categoryGroup struct {
		name    string
		results []CheckResult
	}
	var groups []categoryGroup
	seen := map[string]int{}

	for _, res := range r.Results {
		idx, ok := seen[res.Category]
		if !ok {
			idx = len(groups)
			seen[res.Category] = idx
			groups = append(groups, categoryGroup{name: res.Category})
		}
		groups[idx].results = append(groups[idx].results, res)
	}

	for _, g := range groups {
		fmt.Fprintf(w, "📋 %s\n", g.name)
		for _, res := range g.results {
			icon := res.Status.String()
			if res.Fixed {
				icon = "🔧"
			}
			fmt.Fprintf(w, "  %s %-20s %s\n", icon, res.Name, res.Message)
			if res.Status != StatusPass && !res.Fixed {
				if res.Expected != "" {
					fmt.Fprintf(w, "     Expected: %s\n", res.Expected)
				}
				if res.FixHint != "" {
					fmt.Fprintf(w, "     → %s\n", res.FixHint)
				}
			}
		}
		fmt.Fprintln(w)
	}

	s := r.Summary()
	fmt.Fprintln(w, "================")
	if s.Fixed > 0 {
		fmt.Fprintf(w, "Result: %d passed, %d failed, %d warning(s), %d fixed\n", s.Passed, s.Failed, s.Warnings, s.Fixed)
	} else {
		fmt.Fprintf(w, "Result: %d passed, %d failed, %d warning(s)\n", s.Passed, s.Failed, s.Warnings)
	}
}

// jsonOutput is the shape of the JSON output.
type jsonOutput struct {
	Results []CheckResult `json:"results"`
	Summary Summary       `json:"summary"`
}

// PrintJSON writes JSON formatted output to w.
func (r *Report) PrintJSON(w io.Writer) error {
	out := jsonOutput{
		Results: r.Results,
		Summary: r.Summary(),
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
