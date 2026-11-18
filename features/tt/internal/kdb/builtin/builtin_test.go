package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axsh/tokotachi/features/tt/internal/kdb/paths"
)

type fake struct{ calls []string }

func (f *fake) Status(context.Context, paths.Paths) (json.RawMessage, error) {
	return json.RawMessage(`{"ok":true}`), nil
}
func (f *fake) Distill(context.Context, paths.Paths) error { f.calls = append(f.calls, "distill"); return nil }
func (f *fake) Dump(context.Context, paths.Paths) error    { f.calls = append(f.calls, "dump"); return nil }
func (f *fake) Merge(_ context.Context, _ paths.Paths, prefer string) error {
	f.calls = append(f.calls, "merge:"+prefer)
	return nil
}
func (f *fake) Pack(context.Context, paths.Paths) error { f.calls = append(f.calls, "pack"); return nil }

func TestIsBuiltin(t *testing.T) {
	assert.True(t, IsBuiltin("backlog"))
	assert.False(t, IsBuiltin("designdb"))
}

func TestNotRegistered(t *testing.T) {
	RegisterBacklog(nil, nil)
	assert.False(t, Registered())
	_, err := Status(context.Background(), paths.Paths{})
	assert.ErrorIs(t, err, ErrNotRegistered)
	assert.ErrorIs(t, Distill(context.Background(), paths.Paths{}), ErrNotRegistered)
}

func TestRegisteredOps(t *testing.T) {
	f := &fake{}
	RegisterBacklog(f, f)
	t.Cleanup(func() { RegisterBacklog(nil, nil) })
	require.True(t, Registered())

	ctx := context.Background()
	for _, op := range []string{"distill", "dump", "merge", "pack"} {
		require.NoError(t, Op(ctx, op, paths.Paths{}, "newer"))
	}
	assert.Equal(t, []string{"distill", "dump", "merge:newer", "pack"}, f.calls)
	assert.Error(t, Op(ctx, "bogus", paths.Paths{}, ""))
}
