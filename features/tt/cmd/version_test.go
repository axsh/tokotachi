package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/version"
)

func TestVersionCmd_PrintsEmbedded(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})
	defer rootCmd.SetArgs(nil)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v\n%s", err, buf.String())
	}
	got := buf.String()
	want := "tt " + version.Version
	if !strings.Contains(got, want) {
		t.Fatalf("stdout %q does not contain %q", got, want)
	}
	if !strings.HasSuffix(strings.TrimRight(got, "\r\n"), version.Version) &&
		!strings.Contains(got, want+"\n") && !strings.Contains(got, want+"\r\n") {
		// Prefer exact line shape: "tt vN.N.N"
		if strings.TrimSpace(got) != want {
			t.Fatalf("stdout %q, want line %q", got, want)
		}
	}
}
