package notify

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/oklog/ulid/v2"
)

// GenerateEventID generates a new ULID-based event ID with "E-" prefix.
func GenerateEventID() (string, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now()), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate ULID: %w", err)
	}
	return "E-" + id.String(), nil
}

// ComputeContentHash computes SHA-256 of the canonical representation.
// Input fields: effective_changed_paths + flags + task_summary + raw_notes (sorted).
// Returns "sha256:" + hex digest.
func ComputeContentHash(event *agent.IntakeEvent) string {
	h := sha256.New()

	// Sorted effective changed paths
	paths := make([]string, len(event.EffectiveChangedPaths))
	copy(paths, event.EffectiveChangedPaths)
	sort.Strings(paths)
	for _, p := range paths {
		h.Write([]byte("path:" + p + "\n"))
	}

	// Flags
	if event.Flags != nil {
		if event.Flags.ArchitectureImpact {
			h.Write([]byte("flag:architecture_impact\n"))
		}
		if event.Flags.MemoryRelated {
			h.Write([]byte("flag:memory_related\n"))
		}
		if event.Flags.PromptRelated {
			h.Write([]byte("flag:prompt_related\n"))
		}
		if event.Flags.AgentBehaviorRelated {
			h.Write([]byte("flag:agent_behavior_related\n"))
		}
		if event.Flags.RequiresImmediateAction {
			h.Write([]byte("flag:requires_immediate_action\n"))
		}
	}

	// Task summary
	h.Write([]byte("summary:" + event.TaskSummary + "\n"))

	// Sorted raw notes
	notes := make([]string, len(event.RawNotes))
	copy(notes, event.RawNotes)
	sort.Strings(notes)
	for _, n := range notes {
		h.Write([]byte("note:" + n + "\n"))
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// ComputeContentID computes the coarse fingerprint for semantic grouping.
// Input fields: task_summary + raw_notes + normalized path prefixes.
// Excludes: branch, timestamps, wrapper_version.
// Returns "RAWC-" + hex(sha256).
func ComputeContentID(event *agent.IntakeEvent) string {
	h := sha256.New()

	// Task summary
	h.Write([]byte("summary:" + event.TaskSummary + "\n"))

	// Sorted raw notes
	notes := make([]string, len(event.RawNotes))
	copy(notes, event.RawNotes)
	sort.Strings(notes)
	for _, n := range notes {
		h.Write([]byte("note:" + n + "\n"))
	}

	// Path prefixes (directory parts only, deduplicated and sorted)
	prefixSet := make(map[string]bool)
	for _, p := range event.EffectiveChangedPaths {
		dir := filepath.Dir(p)
		if dir != "." && dir != "" {
			prefixSet[dir] = true
		}
	}
	prefixes := make([]string, 0, len(prefixSet))
	for p := range prefixSet {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)
	for _, p := range prefixes {
		h.Write([]byte("prefix:" + strings.ReplaceAll(p, "\\", "/") + "\n"))
	}

	return "RAWC-" + hex.EncodeToString(h.Sum(nil))
}
