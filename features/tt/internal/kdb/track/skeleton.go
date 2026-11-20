package track

import (
	"fmt"
	"os"
	"path/filepath"
)

// SkeletonYAML returns a registry.yaml template (comments + empty kdbs list).
// It does not include a required backlog entry; backlog is built into tt.
func SkeletonYAML() string {
	return `# Knowledge DB registry (tt kdb-track).
# Builtin backlog is always registered; override it here only when needed.
# *_cmd values are full shell command lines and may include arguments/flags.
version: 1
kdbs:
  # Example external KDB:
  # - id: designdb
  #   module_path: shared/kdb/designdb
  #   corpus_path: shared/kdb/designdb/corpus
  #   status_cmd: "./scripts/design/status.sh --verbose"
  #   distill_cmd: "./scripts/design/distill.sh"
  #   dump_cmd: "./scripts/design/dump.sh"
  #   merge_cmd: "./scripts/design/merge.sh"
  #   pack_cmd: "./scripts/design/pack.sh"
  #
  # Optional override for the builtin backlog:
  # - id: backlog
  #   module_path: shared/kdb/backlog
  #   corpus_path: shared/kdb/backlog/corpus
  #   status_cmd: "tt kdb backlog status"
  #   distill_cmd: "tt kdb backlog distill"
  #   dump_cmd: "tt kdb backlog dump"
  #   merge_cmd: "tt kdb backlog merge"
  #   pack_cmd: "tt kdb backlog pack"
`
}

// WriteSkeleton creates path with SkeletonYAML. Fails if the file already exists.
func WriteSkeleton(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("registry already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(SkeletonYAML())
	return err
}
