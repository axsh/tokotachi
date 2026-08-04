package emitter

import "github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"

// selectAllForTest marks every entity selected. Emitter unit tests construct
// manifests without running ApplyTagSelection, so Selected defaults to false.
func selectAllForTest(resolved *manifest.ResolvedManifest) {
	if resolved == nil {
		return
	}
	for _, list := range resolved.Entities {
		for _, e := range list {
			if e != nil {
				e.Selected = true
			}
		}
	}
}
