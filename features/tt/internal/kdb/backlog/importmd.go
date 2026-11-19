package backlog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

var (
	mdPathRE   = regexp.MustCompile(`(?:^|/)prompts/phases/([^/]+)/branches/([^/]+)/backlogs/(\d+)-([^/]+)\.md$`)
	mdDateRE   = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
	mdHeaderRE = regexp.MustCompile(`^- ([^:：]+)[:：]\s*(.*)$`)
	mdNumRE    = regexp.MustCompile(`^\d+\s+`)
	mdParenRE  = regexp.MustCompile(`[（(].*$`)
)

var mdSectionMap = map[string]string{
	"決定":         "decision",
	"決まった扱い":     "decision",
	"何が問題か":      "problem",
	"今の扱い":       "current_handling",
	"選択肢と影響":     "options",
	"確かめるときの質問":  "questions",
	"決まったらすること":  "after_decision",
	"経緯":         "history",
}

var mdHeaderMap = map[string]string{
	"状態":     "state",
	"種類":     "kind",
	"確かめる相手": "ask_to",
	"いつまでに":  "due",
	"出どころ":   "origin",
}

// ImportResult is the outcome of ImportMarkdown.
type ImportResult struct {
	Imported int      `json:"imported"`
	Files    int      `json:"files"`
	Warnings []string `json:"warnings"`
	Keys     []string `json:"keys"`
}

// NewUUID returns a new random RFC 4122 version 4 uuid string.
func NewUUID() string { return uuid.NewString() }

// ParseMarkdownItems converts legacy backlog markdown files/directories into
// items (bl_json cmd_import_md). Nothing is written.
func ParseMarkdownItems(paths []string, today string) ([]Item, ImportResult, error) {
	warnings := []string{}
	var files []string
	for _, p := range paths {
		st, err := os.Stat(p)
		switch {
		case err == nil && st.IsDir():
			var found []string
			walkErr := filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
					found = append(found, path)
				}
				return nil
			})
			if walkErr != nil {
				return nil, ImportResult{}, bizErr("walk %s: %v", p, walkErr)
			}
			sort.Strings(found)
			skipped := 0
			for _, f := range found {
				if mdPathRE.MatchString(filepath.ToSlash(f)) {
					files = append(files, f)
				} else {
					skipped++
				}
			}
			if skipped > 0 {
				warnings = append(warnings, fmt.Sprintf("skipped %d markdown file(s) outside the backlogs layout under %s", skipped, p))
			}
		case err == nil:
			if !mdPathRE.MatchString(filepath.ToSlash(p)) {
				return nil, ImportResult{}, usageErr("path does not match prompts/phases/<phase>/branches/<branch>/backlogs/<NNN>-<Slug>.md: %s", p)
			}
			files = append(files, p)
		default:
			return nil, ImportResult{}, usageErr("path not found: %s", p)
		}
	}
	if len(files) == 0 {
		return nil, ImportResult{}, bizErr("no backlog markdown files found")
	}
	var items []Item
	seen := map[Key]string{}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, ImportResult{}, bizErr("read %s: %v", f, err)
		}
		it, ok := ParseMarkdownItem(filepath.ToSlash(f), string(raw), today, &warnings)
		if !ok {
			continue
		}
		k := KeyOf(it)
		if prev, dup := seen[k]; dup {
			return nil, ImportResult{}, bizErr("two files map to the same logical key %s: %s and %s", k, prev, f)
		}
		seen[k] = f
		items = append(items, it)
	}
	keys := make([]string, 0, len(items))
	for _, it := range SortItems(items) {
		keys = append(keys, KeyOf(it).String())
	}
	return items, ImportResult{Imported: len(items), Files: len(files), Warnings: warnings, Keys: keys}, nil
}

// ParseMarkdownItem parses one backlog markdown document. relPath must match
// the prompts/phases/<phase>/branches/<branch>/backlogs/<NNN>-<Slug>.md layout.
func ParseMarkdownItem(relPath, text, today string, warnings *[]string) (Item, bool) {
	m := mdPathRE.FindStringSubmatch(relPath)
	if m == nil {
		return Item{}, false
	}
	phase, branch, slug := m[1], m[2], m[4]
	localID := m[3]
	for len(localID) < 3 {
		localID = "0" + localID
	}
	text = strings.TrimPrefix(text, "\ufeff")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	title, headerLines, sections := splitMarkdown(text)
	title = mdNumRE.ReplaceAllString(title, "")
	label := phase + "/" + branch + "/" + localID
	hdr := parseHeader(headerLines, warnings, label)

	it := Item{Phase: phase, Branch: branch, LocalID: localID, Slug: slug, Title: title, UUID: NewUUID()}

	rawState := hdr["状態"]
	state := ""
	for _, s := range States {
		if strings.HasPrefix(rawState, s) {
			state = s
			break
		}
	}
	stateNote := ""
	if state == "" {
		state = StateUnconfirmed
		*warnings = append(*warnings, fmt.Sprintf("%s: unmapped state line %s -> 未確認 (original kept in history)", label, pyRepr(rawState)))
		if rawState != "" {
			stateNote = "- (import) original state line: " + rawState
		}
	}
	it.State = state
	if d := mdDateRE.FindString(rawState); d != "" {
		it.StateDate = StrPtr(d)
	}

	rawKind := hdr["種類"]
	kind := ""
	for _, k := range Kinds {
		if strings.HasPrefix(rawKind, k) {
			kind = k
			break
		}
	}
	if kind == "" {
		for _, k := range Kinds {
			if strings.Contains(rawKind, k) {
				kind = k
				break
			}
		}
	}
	if kind != rawKind {
		*warnings = append(*warnings, fmt.Sprintf("%s: kind %s normalized to %s", label, pyRepr(rawKind), pyRepr(kind)))
	}
	it.Kind = kind
	it.AskTo = hdr["確かめる相手"]
	it.Due = hdr["いつまでに"]
	it.Origin = hdr["出どころ"]

	cols := map[string]*string{
		"decision": &it.Decision, "problem": &it.Problem, "current_handling": &it.CurrentHandling,
		"options": &it.Options, "questions": &it.Questions, "after_decision": &it.AfterDecision,
		"history": &it.History,
	}
	for _, sec := range sections {
		body := trimBlock(sec.lines)
		col, ok := mdSectionMap[sectionName(sec.name)]
		if !ok {
			*warnings = append(*warnings, fmt.Sprintf("%s: unknown section %s appended to problem", label, pyRepr(sec.name)))
			col = "problem"
			body = "## " + sec.name + "\n\n" + body
		}
		dst := cols[col]
		if *dst != "" {
			*dst = *dst + "\n\n" + body
		} else {
			*dst = body
		}
	}
	if stateNote != "" {
		if it.History != "" {
			it.History = it.History + "\n" + stateNote
		} else {
			it.History = stateNote
		}
	}

	dates := mdDateRE.FindAllString(it.History, -1)
	if it.StateDate != nil {
		dates = append(dates, *it.StateDate)
	}
	if len(dates) > 0 {
		sort.Strings(dates)
		it.CreatedAt, it.UpdatedAt = dates[0], dates[len(dates)-1]
	} else {
		it.CreatedAt, it.UpdatedAt = today, today
	}
	return it, true
}

type mdSection struct {
	name  string
	lines []string
}

// splitMarkdown returns the title, header lines (before the first "## "),
// and sections, honoring fenced code blocks.
func splitMarkdown(text string) (string, []string, []*mdSection) {
	title := ""
	var header []string
	var sections []*mdSection
	var cur *mdSection
	inFence := false
	for _, line := range strings.Split(text, "\n") {
		stripped := strings.TrimLeft(line, " \t")
		if !inFence {
			if cur == nil && title == "" && strings.HasPrefix(line, "# ") {
				title = strings.TrimSpace(line[2:])
				continue
			}
			if strings.HasPrefix(line, "## ") {
				cur = &mdSection{name: strings.TrimSpace(line[3:])}
				sections = append(sections, cur)
				continue
			}
		}
		if strings.HasPrefix(stripped, "```") || strings.HasPrefix(stripped, "~~~") {
			inFence = !inFence
		}
		if cur == nil {
			header = append(header, line)
		} else {
			cur.lines = append(cur.lines, line)
		}
	}
	return title, header, sections
}

func parseHeader(lines []string, warnings *[]string, label string) map[string]string {
	fields := map[string]string{}
	var order []string
	key, first := "", ""
	have := false
	var cont []string
	flush := func() {
		if !have {
			return
		}
		value := first
		if len(cont) > 0 {
			body := rstripSpace(strings.Trim(strings.Join(dedent(cont), "\n"), "\n"))
			if first != "" {
				value = first + "\n" + body
			} else {
				value = body
			}
		}
		if _, exists := fields[key]; !exists {
			order = append(order, key)
		}
		fields[key] = value
	}
	for _, line := range lines {
		if m := mdHeaderRE.FindStringSubmatch(line); m != nil {
			flush()
			key, first, cont, have = strings.TrimSpace(m[1]), strings.TrimSpace(m[2]), nil, true
			continue
		}
		if have && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && strings.TrimSpace(line) != "" {
			cont = append(cont, rstripSpace(line))
		}
	}
	flush()
	for _, k := range order {
		if _, ok := mdHeaderMap[k]; !ok {
			*warnings = append(*warnings, fmt.Sprintf("%s: unknown header key ignored: %s", label, k))
		}
	}
	return fields
}

func dedent(lines []string) []string {
	n := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		ind := len(l) - len(strings.TrimLeft(l, " \t"))
		if n < 0 || ind < n {
			n = ind
		}
	}
	if n < 0 {
		n = 0
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			out[i] = ""
		} else {
			out[i] = l[n:]
		}
	}
	return out
}

func trimBlock(lines []string) string {
	return rstripSpace(strings.Trim(strings.Join(lines, "\n"), "\n"))
}

func rstripSpace(s string) string { return strings.TrimRightFunc(s, unicode.IsSpace) }

func sectionName(raw string) string {
	return strings.TrimSpace(mdParenRE.ReplaceAllString(raw, ""))
}

// pyRepr renders s like Python repr() for str values (used in warnings).
func pyRepr(s string) string {
	quote := byte('\'')
	if strings.Contains(s, "'") && !strings.Contains(s, "\"") {
		quote = '"'
	}
	var b strings.Builder
	b.WriteByte(quote)
	for _, r := range s {
		switch {
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r == rune(quote):
			b.WriteByte('\\')
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, `\x%02x`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte(quote)
	return b.String()
}
