package version

import (
	"regexp"
	"testing"
)

func TestVersion_InitialValue(t *testing.T) {
	if Version != "v0.9.1" {
		t.Fatalf("Version = %q, want %q", Version, "v0.9.1")
	}
}

func TestVersion_NonEmpty(t *testing.T) {
	if Version == "" {
		t.Fatal("Version must not be empty")
	}
}

func TestVersion_SemVerWithLeadingV(t *testing.T) {
	re := regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
	if !re.MatchString(Version) {
		t.Fatalf("Version %q does not match vN.N.N", Version)
	}
}
