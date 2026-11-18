package jsonout

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOK(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, OK(&buf, map[string]any{"n": 1, "msg": "日本語"}))
	assert.True(t, bytes.HasSuffix(buf.Bytes(), []byte("\n")))
	var m map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	assert.Equal(t, true, m["ok"])
	assert.Equal(t, "日本語", m["msg"])
}

func TestFail(t *testing.T) {
	var buf bytes.Buffer
	err := Fail(&buf, "tt kdb-track list help", CodeUsage, "bad", map[string]any{"id": "x"})
	var ee *ExitError
	require.True(t, errors.As(err, &ee))
	assert.Equal(t, 2, ee.Code)

	var m map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	assert.Equal(t, false, m["ok"])
	assert.Equal(t, "bad", m["error"])
	assert.Equal(t, "tt kdb-track list help", m["help"])
	assert.Equal(t, "x", m["id"])
}

func TestOK_NilFields(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, OK(&buf, nil))
	assert.JSONEq(t, `{"ok":true}`, buf.String())
}
