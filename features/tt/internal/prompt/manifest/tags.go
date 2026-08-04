package manifest

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// BaselineTag is the implicit selection tag when frontmatter omits tags.
	BaselineTag = "baseline"
	// TagRefsInclude pulls referenced capabilities into the selected set.
	TagRefsInclude = "include"
	// TagRefsStrict requires referenced capabilities to match requested tags.
	TagRefsStrict = "strict"
	// EnvKeyTags is the environment variable for default --tags.
	EnvKeyTags = "TT_TAGS"
	// EnvKeyTagRefs is the environment variable for default --tag-refs.
	EnvKeyTagRefs = "TT_TAG_REFS"
)

var tagNameRegexp = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// IsTaggableKind reports whether kind is subject to --tags filtering.
// policy / procedure / capability / skip are taggable (code_content).
// guard / worker / bundle / target are not.
func IsTaggableKind(kind string) bool {
	switch kind {
	case "policy", "procedure", "capability", "skip":
		return true
	default:
		return false
	}
}

// NormalizeRequestedTags parses a comma-separated tags string.
func NormalizeRequestedTags(raw string) (tags []string, warnings []string, err error) {
	parts := strings.Split(raw, ",")
	seen := make(map[string]bool)
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		if !tagNameRegexp.MatchString(t) {
			return nil, nil, fmt.Errorf("invalid tag name %q: must match %s", t, tagNameRegexp.String())
		}
		if seen[t] {
			warnings = append(warnings, fmt.Sprintf("duplicate tag %q removed", t))
			continue
		}
		seen[t] = true
		tags = append(tags, t)
	}
	if len(tags) == 0 {
		return nil, nil, fmt.Errorf("tags list is empty after normalization")
	}
	return tags, warnings, nil
}

// NormalizeTagRefsMode returns include|strict. Empty input becomes include.
func NormalizeTagRefsMode(raw string) (string, error) {
	m := strings.TrimSpace(raw)
	if m == "" {
		return TagRefsInclude, nil
	}
	switch m {
	case TagRefsInclude, TagRefsStrict:
		return m, nil
	default:
		return "", fmt.Errorf("invalid tag-refs mode %q: must be %q or %q", m, TagRefsInclude, TagRefsStrict)
	}
}

// EffectiveTags returns the effective tag set for an entity.
func EffectiveTags(e *Entity) ([]string, error) {
	if e == nil {
		return nil, fmt.Errorf("entity is nil")
	}
	if e.Raw == nil {
		return []string{BaselineTag}, nil
	}
	val, ok := e.Raw["tags"]
	if !ok {
		return []string{BaselineTag}, nil
	}
	if val == nil {
		return nil, fmt.Errorf("tags must not be null")
	}
	switch v := val.(type) {
	case string:
		t := strings.TrimSpace(v)
		if t == "" {
			return nil, fmt.Errorf("tags string must not be empty")
		}
		if !tagNameRegexp.MatchString(t) {
			return nil, fmt.Errorf("invalid tag name %q: must match %s", t, tagNameRegexp.String())
		}
		return []string{t}, nil
	case []any:
		if len(v) == 0 {
			return nil, fmt.Errorf("tags array must not be empty")
		}
		out := make([]string, 0, len(v))
		seen := make(map[string]bool)
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("tags array elements must be strings, got %T", item)
			}
			s = strings.TrimSpace(s)
			if s == "" {
				return nil, fmt.Errorf("tags array elements must not be empty")
			}
			if !tagNameRegexp.MatchString(s) {
				return nil, fmt.Errorf("invalid tag name %q: must match %s", s, tagNameRegexp.String())
			}
			if seen[s] {
				continue
			}
			seen[s] = true
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("tags must be a string or array of strings, got %T", val)
	}
}

func tagMatched(effective, requested []string) bool {
	req := make(map[string]bool, len(requested))
	for _, t := range requested {
		req[t] = true
	}
	for _, t := range effective {
		if req[t] {
			return true
		}
	}
	return false
}

func referencedCapabilityIDs(e *Entity) []string {
	if e == nil || e.Raw == nil {
		return nil
	}
	usesCaps, ok := e.Raw["uses_capabilities"]
	if !ok {
		return nil
	}
	capList, ok := usesCaps.([]any)
	if !ok {
		return nil
	}
	var ids []string
	for _, cap := range capList {
		s, ok := cap.(string)
		if !ok {
			s = fmt.Sprintf("%v", cap)
		}
		s = strings.TrimSpace(s)
		if s != "" {
			ids = append(ids, s)
		}
	}
	return ids
}

// ApplyTagSelection sets Entity.Selected on all entities according to requested tags and mode.
func ApplyTagSelection(entities []*Entity, requested []string, mode string) []ValidationError {
	var errors []ValidationError
	if len(requested) == 0 {
		requested = []string{BaselineTag}
	}
	mode, err := NormalizeTagRefsMode(mode)
	if err != nil {
		return []ValidationError{{Message: err.Error()}}
	}

	matched := make(map[*Entity]bool, len(entities))
	capByID := make(map[string]*Entity)

	for _, e := range entities {
		if e == nil {
			continue
		}
		if e.Kind == "capability" {
			capByID[e.ID] = e
		}
		if !IsTaggableKind(e.Kind) {
			e.Selected = true
			continue
		}
		effective, err := EffectiveTags(e)
		if err != nil {
			errors = append(errors, ValidationError{
				File:    e.FilePath,
				Message: fmt.Sprintf("tags: %v", err),
			})
			e.Selected = false
			continue
		}
		ok := tagMatched(effective, requested)
		matched[e] = ok
		e.Selected = ok
	}
	if len(errors) > 0 {
		return errors
	}

	switch mode {
	case TagRefsInclude:
		queue := make([]*Entity, 0)
		for _, e := range entities {
			if e != nil && e.Selected {
				queue = append(queue, e)
			}
		}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for _, capID := range referencedCapabilityIDs(cur) {
				cap := capByID[capID]
				if cap == nil || cap.Selected {
					continue
				}
				cap.Selected = true
				queue = append(queue, cap)
			}
		}
	case TagRefsStrict:
		for _, e := range entities {
			if e == nil || !matched[e] {
				continue
			}
			for _, capID := range referencedCapabilityIDs(e) {
				cap := capByID[capID]
				if cap == nil {
					continue
				}
				if !matched[cap] {
					errors = append(errors, ValidationError{
						File: e.FilePath,
						Message: fmt.Sprintf(
							"tag-refs strict: %s %q references capability %q which does not match requested tags",
							e.Kind, e.ID, capID,
						),
					})
				}
			}
		}
	}

	return errors
}
