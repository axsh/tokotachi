package backlog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/axsh/tokotachi/features/tt/internal/kdb/builtin"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/paths"
)

// statusAdapter implements builtin.StatusRunner.
type statusAdapter struct{}

// pipelineAdapter implements builtin.PipelineRunner.
type pipelineAdapter struct{}

var registerOnce sync.Once

// Register wires the backlog runners into builtin so that `tt kdb-track`
// (status-all, systematize) and `tt kdb backlog` use this package.
// It is safe to call more than once.
func Register() {
	registerOnce.Do(func() {
		builtin.RegisterBacklog(statusAdapter{}, pipelineAdapter{})
	})
}

// Status returns the status.sh-shaped JSON {"ok":true,"data":{...}}.
func (statusAdapter) Status(_ context.Context, p paths.Paths) (json.RawMessage, error) {
	fields, err := Ops{P: p}.Status()
	if err != nil {
		return nil, err
	}
	fields["ok"] = true
	return json.Marshal(fields)
}

func (pipelineAdapter) Distill(_ context.Context, p paths.Paths) error {
	_, err := Ops{P: p}.Distill()
	return describeProblems(err)
}

func (pipelineAdapter) Dump(_ context.Context, p paths.Paths) error {
	_, err := Ops{P: p}.Dump()
	return err
}

// Merge fails on conflicts so that systematize stops at the merge stage.
func (pipelineAdapter) Merge(_ context.Context, p paths.Paths, prefer string) error {
	_, err := Ops{P: p}.MergeResult(prefer)
	return err
}

func (pipelineAdapter) Pack(_ context.Context, p paths.Paths) error {
	_, err := Ops{P: p}.Pack()
	return err
}

// describeProblems expands distill problems into the error message.
func describeProblems(err error) error {
	var oe *OpError
	if !errors.As(err, &oe) || oe.Extra == nil {
		return err
	}
	reports, ok := oe.Extra["reports"].([]any)
	if !ok {
		return err
	}
	msg := oe.Msg
	for _, r := range reports {
		var rep DistillReport
		switch v := r.(type) {
		case DistillReport:
			rep = v
		default:
			continue
		}
		for _, pr := range rep.Problems {
			msg += fmt.Sprintf("; %s: %s: %s", rep.Source, pr.Key, pr.Problem)
		}
	}
	return &OpError{Code: oe.Code, Msg: msg, Extra: oe.Extra}
}
