package backlog

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ListOptions are the post-query filters of `list` (bl_json list-filter).
type ListOptions struct {
	Match  string // case-insensitive regexp over title, problem and slug
	Sort   string // one of SortKeys, or ""
	Order  string // asc (default) or desc
	Limit  string // "" or "null" means unlimited; else integer >= 1
	Offset string // "" means 0; else integer >= 0
}

// ListPage is the result of ListFilter.
type ListPage struct {
	Total  int    `json:"total"`
	Limit  *int   `json:"limit"`
	Offset int    `json:"offset"`
	Items  []Item `json:"items"`
}

// ListFilter applies match, sort, order, limit and offset. Total is the count
// before limit and offset are applied.
func ListFilter(items []Item, o ListOptions) (ListPage, error) {
	if o.Match != "" {
		rx, err := regexp.Compile("(?i)" + o.Match)
		if err != nil {
			return ListPage{}, usageErr("invalid --match regular expression: %v", err)
		}
		kept := make([]Item, 0, len(items))
		for _, it := range items {
			if rx.MatchString(it.Title) || rx.MatchString(it.Problem) || rx.MatchString(it.Slug) {
				kept = append(kept, it)
			}
		}
		items = kept
	}
	order := strings.ToLower(o.Order)
	if order == "" {
		order = "asc"
	}
	if order != "asc" && order != "desc" {
		return ListPage{}, usageErr("--order must be asc or desc")
	}
	if o.Sort != "" && !contains(SortKeys, o.Sort) {
		return ListPage{}, usageErr("--sort must be one of: %s", strings.Join(SortKeys, ", "))
	}
	var limit *int
	if o.Limit != "" && o.Limit != "null" {
		n, err := parseNonNegInt("limit", o.Limit, 1)
		if err != nil {
			return ListPage{}, err
		}
		limit = &n
	}
	offset := 0
	if o.Offset != "" {
		n, err := parseNonNegInt("offset", o.Offset, 0)
		if err != nil {
			return ListPage{}, err
		}
		offset = n
	}

	items = SortItems(items)
	switch {
	case o.Sort != "" && o.Sort != "local_id":
		desc := order == "desc"
		sort.SliceStable(items, func(i, j int) bool {
			a, b := sortField(items[i], o.Sort), sortField(items[j], o.Sort)
			if desc {
				return a > b
			}
			return a < b
		})
	case order == "desc":
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
	}
	total := len(items)
	page := []Item{}
	if offset < len(items) {
		end := len(items)
		if limit != nil && offset+*limit < end {
			end = offset + *limit
		}
		page = items[offset:end]
	}
	return ListPage{Total: total, Limit: limit, Offset: offset, Items: page}, nil
}

func sortField(it Item, key string) string {
	switch key {
	case "updated_at":
		return it.UpdatedAt
	case "created_at":
		return it.CreatedAt
	case "state":
		return it.State
	case "title":
		return it.Title
	}
	return it.LocalID
}

func parseNonNegInt(label, value string, minimum int) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, usageErr("--%s must be an integer", label)
	}
	if n < minimum {
		return 0, usageErr("--%s must be >= %d", label, minimum)
	}
	return n, nil
}
