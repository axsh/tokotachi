package backlog

import (
	"errors"
	"fmt"

	"github.com/axsh/tokotachi/features/tt/internal/kdb/jsonout"
)

// ErrMergeConflict is returned by Merge when the same logical key carries
// different content on both sides. The CLI absorbs it (exit 0, restored=false).
var ErrMergeConflict = errors.New("merge conflict: same logical key with different content")

// OpError is a failure that maps to a process exit code and a JSON body.
type OpError struct {
	Code  int
	Msg   string
	Extra map[string]any
}

func (e *OpError) Error() string { return e.Msg }

func usageErr(format string, a ...any) *OpError {
	return &OpError{Code: jsonout.CodeUsage, Msg: fmt.Sprintf(format, a...)}
}

func bizErr(format string, a ...any) *OpError {
	return &OpError{Code: jsonout.CodeBusiness, Msg: fmt.Sprintf(format, a...)}
}

func bizErrExtra(extra map[string]any, format string, a ...any) *OpError {
	return &OpError{Code: jsonout.CodeBusiness, Msg: fmt.Sprintf(format, a...), Extra: extra}
}

// CodeOf returns the exit code carried by err (1 for non-OpError errors).
func CodeOf(err error) int {
	var oe *OpError
	if errors.As(err, &oe) {
		return oe.Code
	}
	return jsonout.CodeBusiness
}
