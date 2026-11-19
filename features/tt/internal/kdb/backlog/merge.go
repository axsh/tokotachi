package backlog

// MergeSummary is the summary of a merge (bl_json cmd_merge output).
type MergeSummary struct {
	DataBefore    int      `json:"data_before"`
	TmpCount      int      `json:"tmp_count"`
	Count         int      `json:"count"`
	Collapsed     int      `json:"collapsed"`
	ConflictCount int      `json:"conflict_count"`
	Conflicts     []string `json:"conflicts"`
	Prefer        string   `json:"prefer"`
}

// MergeItems unions data and tmp items. With prefer "tmp" or "data" the
// preferred side wins for every key present on both sides. After collapsing
// identical duplicates, the result is sorted unless conflicts remain.
func MergeItems(data, tmp []Item, prefer string) ([]Item, MergeSummary, error) {
	if prefer != "" && prefer != "tmp" && prefer != "data" {
		return nil, MergeSummary{}, usageErr("--prefer must be tmp or data")
	}
	tmpKeys := map[Key]bool{}
	for _, it := range tmp {
		tmpKeys[KeyOf(it)] = true
	}
	dataKeys := map[Key]bool{}
	for _, it := range data {
		dataKeys[KeyOf(it)] = true
	}
	merged := make([]Item, 0, len(data)+len(tmp))
	for _, it := range data {
		if prefer == "tmp" && tmpKeys[KeyOf(it)] {
			continue
		}
		merged = append(merged, it)
	}
	for _, it := range tmp {
		if prefer == "data" && dataKeys[KeyOf(it)] {
			continue
		}
		merged = append(merged, it)
	}
	merged, collapsed, conflicts := Collapse(merged)
	if len(conflicts) == 0 {
		merged = SortItems(merged)
	}
	if merged == nil {
		merged = []Item{}
	}
	sum := MergeSummary{
		DataBefore: len(data), TmpCount: len(tmp), Count: len(merged), Collapsed: collapsed,
		ConflictCount: len(conflicts), Conflicts: keyStrings(conflicts), Prefer: prefer,
	}
	return merged, sum, nil
}

// MergeFiles merges the tmp dump and the data array file into outPath (the
// data file). The merged array is written even when conflicts remain; in that
// case ErrMergeConflict is returned together with the summary.
func MergeFiles(tmpPath, dataPath, outPath, prefer string) (MergeSummary, error) {
	tmpItems, err := ReadItems(tmpPath)
	if err != nil {
		return MergeSummary{}, err
	}
	dataItems, err := ReadItems(dataPath)
	if err != nil {
		return MergeSummary{}, err
	}
	merged, sum, err := MergeItems(dataItems, tmpItems, prefer)
	if err != nil {
		return MergeSummary{}, err
	}
	if err := WriteItems(outPath, merged); err != nil {
		return MergeSummary{}, err
	}
	if sum.ConflictCount > 0 {
		return sum, ErrMergeConflict
	}
	return sum, nil
}
