// Package jsonout writes machine-readable success/failure JSON for kdb commands.
package jsonout

import (
	"encoding/json"
	"fmt"
	"io"
)

// Exit codes used by kdb commands.
const (
	CodeBusiness      = 1
	CodeUsage         = 2
	CodeMergeConflict = 3
)

// ExitError carries a process exit code through cobra's RunE.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit %d", e.Code)
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }

// OK writes {"ok":true, ...fields} as compact JSON followed by a newline.
func OK(w io.Writer, fields map[string]any) error {
	m := map[string]any{}
	for k, v := range fields {
		m[k] = v
	}
	m["ok"] = true
	return writeJSON(w, m)
}

// Fail writes {"ok":false,"error":msg,"help":helpHint,...extra} and returns
// an *ExitError carrying code. The caller decides how to terminate.
func Fail(w io.Writer, helpHint string, code int, msg string, extra map[string]any) error {
	m := map[string]any{}
	for k, v := range extra {
		m[k] = v
	}
	m["ok"] = false
	m["error"] = msg
	m["help"] = helpHint
	if err := writeJSON(w, m); err != nil {
		return err
	}
	return &ExitError{Code: code, Err: fmt.Errorf("%s", msg)}
}

func writeJSON(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(append(data, '\n'))
	return err
}
