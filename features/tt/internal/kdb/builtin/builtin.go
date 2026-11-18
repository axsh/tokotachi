// Package builtin holds hooks for KDBs implemented inside tt (currently backlog only).
// Other KDBs are delegated via their registry *_cmd entries.
package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/axsh/tokotachi/features/tt/internal/kdb/paths"
)

// BacklogID is the only builtin KDB id.
const BacklogID = "backlog"

// ErrNotRegistered is returned when the builtin runner has not been wired yet.
var ErrNotRegistered = errors.New("builtin backlog runner is not registered (backlog support is not wired in this build)")

// IsBuiltin reports whether id is implemented inside tt.
func IsBuiltin(id string) bool { return id == BacklogID }

// StatusRunner is implemented by the backlog package.
type StatusRunner interface {
	Status(ctx context.Context, p paths.Paths) (json.RawMessage, error)
}

// PipelineRunner is implemented by the backlog package.
type PipelineRunner interface {
	Distill(ctx context.Context, p paths.Paths) error
	Dump(ctx context.Context, p paths.Paths) error
	Merge(ctx context.Context, p paths.Paths, prefer string) error
	Pack(ctx context.Context, p paths.Paths) error
}

var (
	mu     sync.RWMutex
	status StatusRunner
	pipe   PipelineRunner
)

// RegisterBacklog installs the backlog runners. Passing nils unregisters them.
func RegisterBacklog(s StatusRunner, p PipelineRunner) {
	mu.Lock()
	defer mu.Unlock()
	status, pipe = s, p
}

// Registered reports whether both runners are installed.
func Registered() bool {
	mu.RLock()
	defer mu.RUnlock()
	return status != nil && pipe != nil
}

func getStatus() (StatusRunner, error) {
	mu.RLock()
	defer mu.RUnlock()
	if status == nil {
		return nil, ErrNotRegistered
	}
	return status, nil
}

func getPipe() (PipelineRunner, error) {
	mu.RLock()
	defer mu.RUnlock()
	if pipe == nil {
		return nil, ErrNotRegistered
	}
	return pipe, nil
}

// Status runs the registered backlog status runner.
func Status(ctx context.Context, p paths.Paths) (json.RawMessage, error) {
	s, err := getStatus()
	if err != nil {
		return nil, err
	}
	return s.Status(ctx, p)
}

// Distill runs the registered backlog distill step.
func Distill(ctx context.Context, p paths.Paths) error {
	r, err := getPipe()
	if err != nil {
		return err
	}
	return r.Distill(ctx, p)
}

// Dump runs the registered backlog dump step.
func Dump(ctx context.Context, p paths.Paths) error {
	r, err := getPipe()
	if err != nil {
		return err
	}
	return r.Dump(ctx, p)
}

// Merge runs the registered backlog merge step.
func Merge(ctx context.Context, p paths.Paths, prefer string) error {
	r, err := getPipe()
	if err != nil {
		return err
	}
	return r.Merge(ctx, p, prefer)
}

// Pack runs the registered backlog pack step.
func Pack(ctx context.Context, p paths.Paths) error {
	r, err := getPipe()
	if err != nil {
		return err
	}
	return r.Pack(ctx, p)
}

// Op runs a single named pipeline operation (distill|dump|merge|pack).
func Op(ctx context.Context, op string, p paths.Paths, prefer string) error {
	switch op {
	case "distill":
		return Distill(ctx, p)
	case "dump":
		return Dump(ctx, p)
	case "merge":
		return Merge(ctx, p, prefer)
	case "pack":
		return Pack(ctx, p)
	}
	return fmt.Errorf("unknown pipeline op %q", op)
}
