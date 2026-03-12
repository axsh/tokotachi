package scaffold

import (
	"fmt"
	"io"
	"strings"
)

// PrintPlan outputs the execution plan in a human-readable format.
func PrintPlan(plan *Plan, w io.Writer) {
	// If there are dependency plans, show each sub-plan with a header
	if len(plan.DependencyPlans) > 0 {
		fmt.Fprintln(w, "\nDependency chain:")
		for i, dp := range plan.DependencyPlans {
			fmt.Fprintf(w, "\n  [%d/%d] %s/%s\n",
				i+1, len(plan.DependencyPlans), dp.Entry.Category, dp.Entry.Name)
			if dp.SubPlan != nil {
				printSubPlan(dp.SubPlan, w, "    ")
			}
		}
		fmt.Fprintln(w)
		return
	}

	// Single scaffold plan (no dependencies)
	printSinglePlan(plan, w)
}

// printSinglePlan outputs a single scaffold plan (original format).
func printSinglePlan(plan *Plan, w io.Writer) {
	fmt.Fprintf(w, "\nScaffold: %q\n\n", plan.ScaffoldName)

	totalCreate := len(plan.FilesToCreate)
	totalSkip := len(plan.FilesToSkip)
	totalModify := len(plan.FilesToModify)

	if totalCreate > 0 {
		fmt.Fprintln(w, "Files to create:")
		for _, f := range plan.FilesToCreate {
			fmt.Fprintf(w, "  [CREATE] %s\n", f.Path)
		}
		fmt.Fprintln(w)
	}

	if totalSkip > 0 {
		fmt.Fprintln(w, "Files to skip (already exist):")
		for _, f := range plan.FilesToSkip {
			fmt.Fprintf(w, "  [SKIP] %s (policy: %s)\n", f.Path, f.ConflictPolicy)
		}
		fmt.Fprintln(w)
	}

	if totalModify > 0 {
		fmt.Fprintln(w, "Files to modify:")
		for _, f := range plan.FilesToModify {
			fmt.Fprintf(w, "  [%s] %s (policy: %s)\n",
				strings.ToUpper(f.Action), f.Path, f.ConflictPolicy)
		}
		fmt.Fprintln(w)
	}

	if len(plan.Warnings) > 0 {
		fmt.Fprintln(w, "WARNING:")
		for _, w2 := range plan.Warnings {
			fmt.Fprintf(w, "  ⚠ %s\n", w2)
		}
		fmt.Fprintln(w)
	}

	// Post-actions section
	postActionCount := len(plan.PostActions.GitignoreEntries) + len(plan.PermissionActions)
	hasPostActions := postActionCount > 0

	if hasPostActions {
		fmt.Fprintln(w, "Post-actions:")
		for _, entry := range plan.PostActions.GitignoreEntries {
			fmt.Fprintf(w, "  [GITIGNORE] Add %q to .gitignore\n", entry)
		}
		for _, pa := range plan.PermissionActions {
			fmt.Fprintf(w, "  [CHMOD %s] %s\n", pa.Mode, pa.Path)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "Summary: %d to create, %d to skip, %d to modify, %d post-actions\n",
		totalCreate, totalSkip, totalModify, postActionCount)
}

// printSubPlan outputs a sub-plan with a given indent prefix.
func printSubPlan(plan *Plan, w io.Writer, indent string) {
	totalCreate := len(plan.FilesToCreate)
	totalSkip := len(plan.FilesToSkip)
	totalModify := len(plan.FilesToModify)

	if totalCreate > 0 {
		for _, f := range plan.FilesToCreate {
			fmt.Fprintf(w, "%s[CREATE] %s\n", indent, f.Path)
		}
	}

	if totalSkip > 0 {
		for _, f := range plan.FilesToSkip {
			fmt.Fprintf(w, "%s[SKIP]   %s\n", indent, f.Path)
		}
	}

	if totalModify > 0 {
		for _, f := range plan.FilesToModify {
			fmt.Fprintf(w, "%s[%s] %s\n", indent, strings.ToUpper(f.Action), f.Path)
		}
	}

	if len(plan.Warnings) > 0 {
		for _, w2 := range plan.Warnings {
			fmt.Fprintf(w, "%s⚠ %s\n", indent, w2)
		}
	}

	fmt.Fprintf(w, "%sSummary: %d create, %d skip, %d modify\n",
		indent, totalCreate, totalSkip, totalModify)
}
