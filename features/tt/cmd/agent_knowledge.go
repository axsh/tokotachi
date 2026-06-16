package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/agent/knowledge"
)

var agentKnowledgeCmd = &cobra.Command{
	Use:   "knowledge",
	Short: "Manage far-knowledge categories",
}

// --- add ---

var agentKnowledgeAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Create a new category and add a knowledge file",
	RunE:  runAgentKnowledgeAdd,
}

var (
	knowledgeAddCategoryPath string
	knowledgeAddTitle        string
	knowledgeAddDescription  string
	knowledgeAddContentFile  string
	knowledgeAddSourceEvents string
)

// --- append ---

var agentKnowledgeAppendCmd = &cobra.Command{
	Use:   "append",
	Short: "Add a knowledge file to an existing category",
	RunE:  runAgentKnowledgeAppend,
}

var (
	knowledgeAppendCategoryPath string
	knowledgeAppendTitle        string
	knowledgeAppendContentFile  string
	knowledgeAppendSourceEvents string
)

// --- list ---

var agentKnowledgeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List knowledge categories",
	RunE:  runAgentKnowledgeList,
}

// --- split ---

var agentKnowledgeSplitCmd = &cobra.Command{
	Use:   "split [category-path]",
	Short: "Split a category into subcategories",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentKnowledgeSplit,
}

var (
	knowledgeSplitInto []string
	knowledgeSplitPlan string
)

// --- merge ---

var agentKnowledgeMergeCmd = &cobra.Command{
	Use:   "merge [category-path-1] [category-path-2] ...",
	Short: "Merge categories into one",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runAgentKnowledgeMerge,
}

var (
	knowledgeMergeInto string
	knowledgeMergePlan string
)

// --- rename ---

var agentKnowledgeRenameCmd = &cobra.Command{
	Use:   "rename [old-path] [new-path]",
	Short: "Rename a category",
	Args:  cobra.ExactArgs(2),
	RunE:  runAgentKnowledgeRename,
}

var knowledgeRenameTitle string

// --- move ---

var agentKnowledgeMoveCmd = &cobra.Command{
	Use:   "move",
	Short: "Move a knowledge file to a different category",
	RunE:  runAgentKnowledgeMove,
}

var (
	knowledgeMoveFrom string
	knowledgeMoveTo   string
)

func init() {
	// add flags
	agentKnowledgeAddCmd.Flags().StringVar(&knowledgeAddCategoryPath, "category-path", "", "Category path (e.g., error-handling)")
	agentKnowledgeAddCmd.Flags().StringVar(&knowledgeAddTitle, "title", "", "Knowledge title")
	agentKnowledgeAddCmd.Flags().StringVar(&knowledgeAddDescription, "description", "", "Category description")
	agentKnowledgeAddCmd.Flags().StringVar(&knowledgeAddContentFile, "content-file", "", "Path to content markdown file")
	agentKnowledgeAddCmd.Flags().StringVar(&knowledgeAddSourceEvents, "source-events", "", "Comma-separated source event IDs")
	_ = agentKnowledgeAddCmd.MarkFlagRequired("category-path")
	_ = agentKnowledgeAddCmd.MarkFlagRequired("title")
	_ = agentKnowledgeAddCmd.MarkFlagRequired("content-file")

	// append flags
	agentKnowledgeAppendCmd.Flags().StringVar(&knowledgeAppendCategoryPath, "category-path", "", "Category path")
	agentKnowledgeAppendCmd.Flags().StringVar(&knowledgeAppendTitle, "title", "", "Knowledge title")
	agentKnowledgeAppendCmd.Flags().StringVar(&knowledgeAppendContentFile, "content-file", "", "Path to content markdown file")
	agentKnowledgeAppendCmd.Flags().StringVar(&knowledgeAppendSourceEvents, "source-events", "", "Comma-separated source event IDs")
	_ = agentKnowledgeAppendCmd.MarkFlagRequired("category-path")
	_ = agentKnowledgeAppendCmd.MarkFlagRequired("title")
	_ = agentKnowledgeAppendCmd.MarkFlagRequired("content-file")

	// split flags
	agentKnowledgeSplitCmd.Flags().StringArrayVar(&knowledgeSplitInto, "into", nil, "Target subcategory names (can be repeated)")
	agentKnowledgeSplitCmd.Flags().StringVar(&knowledgeSplitPlan, "plan", "", "Path to split plan JSON file")
	_ = agentKnowledgeSplitCmd.MarkFlagRequired("into")
	_ = agentKnowledgeSplitCmd.MarkFlagRequired("plan")

	// merge flags
	agentKnowledgeMergeCmd.Flags().StringVar(&knowledgeMergeInto, "into", "", "Target category path")
	agentKnowledgeMergeCmd.Flags().StringVar(&knowledgeMergePlan, "plan", "", "Path to merge plan JSON file")
	_ = agentKnowledgeMergeCmd.MarkFlagRequired("into")
	_ = agentKnowledgeMergeCmd.MarkFlagRequired("plan")

	// rename flags
	agentKnowledgeRenameCmd.Flags().StringVar(&knowledgeRenameTitle, "title", "", "New category title")
	_ = agentKnowledgeRenameCmd.MarkFlagRequired("title")

	// move flags
	agentKnowledgeMoveCmd.Flags().StringVar(&knowledgeMoveFrom, "from", "", "Source knowledge file path (relative to knowledge root)")
	agentKnowledgeMoveCmd.Flags().StringVar(&knowledgeMoveTo, "to", "", "Target category path")
	_ = agentKnowledgeMoveCmd.MarkFlagRequired("from")
	_ = agentKnowledgeMoveCmd.MarkFlagRequired("to")

	// Register all subcommands
	agentKnowledgeCmd.AddCommand(agentKnowledgeAddCmd)
	agentKnowledgeCmd.AddCommand(agentKnowledgeAppendCmd)
	agentKnowledgeCmd.AddCommand(agentKnowledgeListCmd)
	agentKnowledgeCmd.AddCommand(agentKnowledgeSplitCmd)
	agentKnowledgeCmd.AddCommand(agentKnowledgeMergeCmd)
	agentKnowledgeCmd.AddCommand(agentKnowledgeRenameCmd)
	agentKnowledgeCmd.AddCommand(agentKnowledgeMoveCmd)

	agentCmd.AddCommand(agentKnowledgeCmd)
}

func newKnowledgeStore() *knowledge.Store {
	rootDir := filepath.Join("prompts", "memory", "knowledge")
	return knowledge.NewStore(rootDir)
}

func parseSourceEvents(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func runAgentKnowledgeAdd(cmd *cobra.Command, args []string) error {
	s := newKnowledgeStore()
	events := parseSourceEvents(knowledgeAddSourceEvents)

	if err := s.Add(knowledgeAddCategoryPath, knowledgeAddTitle, knowledgeAddDescription, knowledgeAddContentFile, events); err != nil {
		return fmt.Errorf("failed to add knowledge: %w", err)
	}
	fmt.Printf("Knowledge '%s' added to category '%s'\n", knowledgeAddTitle, knowledgeAddCategoryPath)
	return nil
}

func runAgentKnowledgeAppend(cmd *cobra.Command, args []string) error {
	s := newKnowledgeStore()
	events := parseSourceEvents(knowledgeAppendSourceEvents)

	if err := s.Append(knowledgeAppendCategoryPath, knowledgeAppendTitle, knowledgeAppendContentFile, events); err != nil {
		return fmt.Errorf("failed to append knowledge: %w", err)
	}
	fmt.Printf("Knowledge '%s' appended to category '%s'\n", knowledgeAppendTitle, knowledgeAppendCategoryPath)
	return nil
}

func runAgentKnowledgeList(cmd *cobra.Command, args []string) error {
	s := newKnowledgeStore()

	categories, err := s.List()
	if err != nil {
		return fmt.Errorf("failed to list knowledge: %w", err)
	}

	if len(categories) == 0 {
		fmt.Println("No knowledge categories found.")
		return nil
	}

	data, err := json.MarshalIndent(categories, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal categories: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func runAgentKnowledgeSplit(cmd *cobra.Command, args []string) error {
	s := newKnowledgeStore()
	categoryPath := args[0]

	if err := s.Split(categoryPath, knowledgeSplitInto, knowledgeSplitPlan); err != nil {
		return fmt.Errorf("failed to split category: %w", err)
	}
	fmt.Printf("Category '%s' split into %v\n", categoryPath, knowledgeSplitInto)
	return nil
}

func runAgentKnowledgeMerge(cmd *cobra.Command, args []string) error {
	s := newKnowledgeStore()

	if err := s.Merge(args, knowledgeMergeInto, knowledgeMergePlan); err != nil {
		return fmt.Errorf("failed to merge categories: %w", err)
	}
	fmt.Printf("Categories merged into '%s'\n", knowledgeMergeInto)
	return nil
}

func runAgentKnowledgeRename(cmd *cobra.Command, args []string) error {
	s := newKnowledgeStore()
	oldPath := args[0]
	newPath := args[1]

	if err := s.Rename(oldPath, newPath, knowledgeRenameTitle); err != nil {
		return fmt.Errorf("failed to rename category: %w", err)
	}
	fmt.Printf("Category '%s' renamed to '%s'\n", oldPath, newPath)
	return nil
}

func runAgentKnowledgeMove(cmd *cobra.Command, args []string) error {
	s := newKnowledgeStore()

	if err := s.Move(knowledgeMoveFrom, knowledgeMoveTo); err != nil {
		return fmt.Errorf("failed to move knowledge file: %w", err)
	}
	fmt.Printf("Knowledge file '%s' moved to '%s'\n", knowledgeMoveFrom, knowledgeMoveTo)
	return nil
}
