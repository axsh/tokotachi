package backlog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var chunkNameRE = regexp.MustCompile(`^chunk-\d{4}\.json$`)

// Chunk is one corpus chunk file (relative path, bytes, sha256, item count).
type Chunk struct {
	Path  string
	Data  []byte
	SHA   string
	Count int
}

// ManifestChunk is a chunk entry of manifest.json.
type ManifestChunk struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Count  int    `json:"count"`
	SHA256 string `json:"sha256"`
}

// Manifest is the corpus manifest.json model.
type Manifest struct {
	KDB           string          `json:"kdb"`
	FormatVersion int             `json:"format_version"`
	ContentHash   string          `json:"content_hash"`
	GeneratedAt   string          `json:"generated_at"`
	ChunkSize     int             `json:"chunk_size"`
	Chunks        []ManifestChunk `json:"chunks"`
}

// PackResult is the outcome of Pack.
type PackResult struct {
	ItemCount   int    `json:"item_count"`
	ChunkCount  int    `json:"chunk_count"`
	ContentHash string `json:"content_hash"`
	Corpus      string `json:"corpus"`
}

// UnpackResult is the outcome of Unpack.
type UnpackResult struct {
	ItemCount   int    `json:"item_count"`
	ContentHash string `json:"content_hash"`
	Out         string `json:"out"`
}

// SHA256Hex returns the lowercase hex SHA-256 of data.
func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// SHA256LF hashes data with CRLF folded to LF so autocrlf checkouts still verify.
func SHA256LF(data []byte) string {
	return SHA256Hex(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")))
}

// BuildChunks sorts items, splits them into ChunkSize groups, and returns the
// chunks plus the content hash (SHA-256 of the concatenated chunk sha hex
// strings as ASCII).
func BuildChunks(items []Item) ([]Chunk, string) {
	sorted := SortItems(items)
	var chunks []Chunk
	var shas strings.Builder
	for i := 0; i < len(sorted); i += ChunkSize {
		end := i + ChunkSize
		if end > len(sorted) {
			end = len(sorted)
		}
		part := sorted[i:end]
		data := CanonicalArrayBytes(part)
		c := Chunk{
			Path:  fmt.Sprintf("items/chunk-%04d.json", i/ChunkSize),
			Data:  data,
			SHA:   SHA256LF(data),
			Count: len(part),
		}
		chunks = append(chunks, c)
		shas.WriteString(c.SHA)
	}
	return chunks, SHA256Hex([]byte(shas.String()))
}

// ContentHashOf returns the corpus content hash of items.
func ContentHashOf(items []Item) string {
	_, h := BuildChunks(items)
	return h
}

// NowRFC3339 returns the current UTC time as RFC 3339 (seconds precision).
func NowRFC3339() string { return time.Now().UTC().Format("2006-01-02T15:04:05Z") }

// LoadManifest reads corpus/manifest.json; (nil, nil) when it does not exist.
func LoadManifest(corpusDir string) (*Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(corpusDir, "manifest.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, bizErr("read manifest.json: %v", err)
	}
	var m Manifest
	if err := json.Unmarshal(bytes.TrimPrefix(raw, utf8BOM), &m); err != nil {
		return nil, bizErr("invalid manifest.json: %v", err)
	}
	return &m, nil
}

// manifestBytes renders the manifest like Python json.dumps(indent=2, ensure_ascii=False).
func manifestBytes(m Manifest) []byte {
	var b strings.Builder
	b.WriteString("{\n")
	field := func(name, jsonVal string, last bool) {
		b.WriteString("  ")
		writePyString(&b, name)
		b.WriteString(": ")
		b.WriteString(jsonVal)
		if !last {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	quote := func(s string) string {
		var sb strings.Builder
		writePyString(&sb, s)
		return sb.String()
	}
	field("kdb", quote(m.KDB), false)
	field("format_version", fmt.Sprint(m.FormatVersion), false)
	field("content_hash", quote(m.ContentHash), false)
	field("generated_at", quote(m.GeneratedAt), false)
	field("chunk_size", fmt.Sprint(m.ChunkSize), false)
	if len(m.Chunks) == 0 {
		field("chunks", "[]", true)
	} else {
		b.WriteString("  \"chunks\": [\n")
		for i, c := range m.Chunks {
			b.WriteString("    {\n")
			b.WriteString("      \"path\": " + quote(c.Path) + ",\n")
			b.WriteString("      \"kind\": " + quote(c.Kind) + ",\n")
			b.WriteString("      \"count\": " + fmt.Sprint(c.Count) + ",\n")
			b.WriteString("      \"sha256\": " + quote(c.SHA256) + "\n")
			b.WriteString("    }")
			if i < len(m.Chunks)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString("  ]\n")
	}
	b.WriteString("}\n")
	return []byte(b.String())
}

// Pack writes items as sorted corpus chunks plus manifest.json under corpusDir.
// Duplicate logical keys are rejected. generated_at is preserved when the
// content hash is unchanged so an unchanged pack produces no Git diff.
func Pack(items []Item, corpusDir string) (PackResult, error) {
	_, collapsed, conflicts := Collapse(items)
	if collapsed > 0 || len(conflicts) > 0 {
		return PackResult{}, bizErrExtra(map[string]any{
			"conflicts": keyStrings(conflicts), "collapsed": collapsed,
		}, "duplicate logical keys in input; run distill first")
	}
	chunks, hash := BuildChunks(items)
	itemsDir := filepath.Join(corpusDir, "items")
	if err := os.MkdirAll(itemsDir, 0o755); err != nil {
		return PackResult{}, bizErr("create %s: %v", itemsDir, err)
	}
	wanted := map[string]bool{}
	for _, c := range chunks {
		wanted[path.Base(c.Path)] = true
		if err := writeFileBytes(filepath.Join(corpusDir, filepath.FromSlash(c.Path)), c.Data); err != nil {
			return PackResult{}, err
		}
	}
	entries, err := os.ReadDir(itemsDir)
	if err != nil {
		return PackResult{}, bizErr("read %s: %v", itemsDir, err)
	}
	for _, e := range entries {
		if chunkNameRE.MatchString(e.Name()) && !wanted[e.Name()] {
			if err := os.Remove(filepath.Join(itemsDir, e.Name())); err != nil {
				return PackResult{}, bizErr("remove stale chunk %s: %v", e.Name(), err)
			}
		}
	}
	old, err := LoadManifest(corpusDir)
	if err != nil {
		return PackResult{}, err
	}
	generatedAt := NowRFC3339()
	if old != nil && old.ContentHash == hash && old.GeneratedAt != "" {
		generatedAt = old.GeneratedAt
	}
	m := Manifest{
		KDB: "backlog", FormatVersion: FormatVersion, ContentHash: hash,
		GeneratedAt: generatedAt, ChunkSize: ChunkSize, Chunks: []ManifestChunk{},
	}
	for _, c := range chunks {
		m.Chunks = append(m.Chunks, ManifestChunk{Path: c.Path, Kind: "items", Count: c.Count, SHA256: c.SHA})
	}
	if err := writeFileBytes(filepath.Join(corpusDir, "manifest.json"), manifestBytes(m)); err != nil {
		return PackResult{}, err
	}
	return PackResult{ItemCount: len(items), ChunkCount: len(chunks), ContentHash: hash, Corpus: corpusDir}, nil
}

// ReadCorpus loads and verifies a corpus (manifest, chunk sha256, counts, content hash).
func ReadCorpus(corpusDir string) ([]Item, string, error) {
	m, err := LoadManifest(corpusDir)
	if err != nil {
		return nil, "", err
	}
	if m == nil {
		return nil, "", bizErr("corpus manifest not found: %s", filepath.Join(corpusDir, "manifest.json"))
	}
	if m.KDB != "backlog" {
		return nil, "", bizErr("manifest kdb is not backlog")
	}
	items := []Item{}
	var shas strings.Builder
	for _, ch := range m.Chunks {
		if !safeChunkPath(ch.Path) {
			return nil, "", bizErr("unsafe chunk path: %s", ch.Path)
		}
		data, err := os.ReadFile(filepath.Join(corpusDir, filepath.FromSlash(ch.Path)))
		if os.IsNotExist(err) {
			return nil, "", bizErr("missing chunk: %s", ch.Path)
		}
		if err != nil {
			return nil, "", bizErr("read chunk %s: %v", ch.Path, err)
		}
		sha := SHA256LF(data)
		if sha != ch.SHA256 {
			return nil, "", bizErr("chunk sha256 mismatch: %s", ch.Path)
		}
		part, err := ParseItems(data, ch.Path)
		if err != nil {
			return nil, "", err
		}
		if len(part) != ch.Count {
			return nil, "", bizErr("chunk count mismatch: %s", ch.Path)
		}
		shas.WriteString(sha)
		items = append(items, part...)
	}
	want := SHA256Hex([]byte(shas.String()))
	if want != m.ContentHash {
		return nil, "", bizErr("manifest content_hash does not match chunks")
	}
	return items, want, nil
}

// Unpack expands a verified corpus into a JSON array file at outJSON.
func Unpack(corpusDir, outJSON string) (UnpackResult, error) {
	items, hash, err := ReadCorpus(corpusDir)
	if err != nil {
		return UnpackResult{}, err
	}
	if err := WriteItems(outJSON, items); err != nil {
		return UnpackResult{}, err
	}
	return UnpackResult{ItemCount: len(items), ContentHash: hash, Out: outJSON}, nil
}

// safeChunkPath rejects absolute paths and parent traversal in manifest chunk paths.
func safeChunkPath(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") || strings.Contains(p, "\\") || strings.Contains(p, ":") {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." || seg == "" {
			return false
		}
	}
	return true
}
