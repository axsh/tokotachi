package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJaccard_Basic(t *testing.T) {
	a := map[string]struct{}{"ab": {}, "bc": {}}
	b := map[string]struct{}{"bc": {}, "cd": {}}
	assert.InDelta(t, 1.0/3.0, Jaccard(a, b), 1e-9)
	assert.Equal(t, 1.0, Jaccard(nil, nil))
	assert.Equal(t, 0.0, Jaccard(a, nil))
}

func TestSimilarity_Identical(t *testing.T) {
	s := "hello world capability"
	assert.Equal(t, 1.0, SimilarityScore(s, s, DefaultNGramN))
}

func TestSimilarity_FixtureDispersion(t *testing.T) {
	base := `---
apiVersion: agent.meta/v1
kind: capability
id: sample-skill
title: Sample Skill
description: does useful things
body: inline
tags:
  - baseline
---

# Sample Skill

This skill walks through scaffold updates and merge policies for manifests.
Prefer keeping project-specific notes while absorbing upstream improvements.
`

	minorEdit := strings.Replace(base, "does useful things", "does useful things for agents", 1)
	renamedBody := strings.Replace(base, "id: sample-skill", "id: sample-skill-renamed", 1)
	renamedBody = strings.Replace(renamedBody, "# Sample Skill", "# Sample Skill Renamed", 1)

	unrelated := `---
apiVersion: agent.meta/v1
kind: capability
id: database-migrate
title: Database Migrate
description: migrates schema versions
body: inline
tags:
  - baseline
---

# Database Migrate

Run SQL migrations against PostgreSQL and record applied versions in a ledger table.
Never skip checksum validation for production databases.
`

	boilerplateOnly := `---
apiVersion: agent.meta/v1
kind: capability
id: other-thing
title: Other Thing
description: unrelated topic
body: inline
tags:
  - baseline
---

# Other Thing

Completely different body about logging levels and stderr writers.
`

	type pair struct {
		name  string
		left  string
		right string
		want  string // high | low
	}
	pairs := []pair{
		{name: "minor-edit", left: base, right: minorEdit, want: "high"},
		{name: "rename-equiv", left: base, right: renamedBody, want: "high"},
		{name: "unrelated", left: base, right: unrelated, want: "low"},
		{name: "boilerplate", left: base, right: boilerplateOnly, want: "low"},
	}

	candidates := []int{3, 4, 5}
	var bestN int
	bestMargin := -1.0
	for _, n := range candidates {
		highMin := 1.0
		lowMax := 0.0
		for _, p := range pairs {
			score := SimilarityScore(p.left, p.right, n)
			t.Logf("n=%d %s=%.4f", n, p.name, score)
			if p.want == "high" {
				if score < highMin {
					highMin = score
				}
			} else if score > lowMax {
				lowMax = score
			}
		}
		margin := highMin - lowMax
		t.Logf("n=%d highMin=%.4f lowMax=%.4f margin=%.4f", n, highMin, lowMax, margin)
		if margin > bestMargin {
			bestMargin = margin
			bestN = n
		}
	}
	require.Greater(t, bestMargin, 0.0, "no n produced positive dispersion (high > low)")
	assert.Equal(t, bestN, DefaultNGramN,
		"DefaultNGramN must match fixture-selected n=%d margin=%.4f", bestN, bestMargin)

	highMin := 1.0
	lowMax := 0.0
	for _, p := range pairs {
		score := SimilarityScore(p.left, p.right, DefaultNGramN)
		if p.want == "high" && score < highMin {
			highMin = score
		}
		if p.want == "low" && score > lowMax {
			lowMax = score
		}
	}
	assert.Greater(t, highMin, lowMax)
}
