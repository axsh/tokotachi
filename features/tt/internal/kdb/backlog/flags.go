package backlog

import "strings"

// Flags holds parsed `--name value` pairs. A key is present only when the
// flag was given (an empty value is still "set").
type Flags map[string]string

// Has reports whether the flag was given.
func (f Flags) Has(name string) bool { _, ok := f[name]; return ok }

// Get returns the flag value ("" when absent).
func (f Flags) Get(name string) string { return f[name] }

// ParseFlags parses `--name value` and `--name=value` arguments (the same
// grammar as the former bl_parse_flags). Non-flag arguments are returned as
// positionals. Unknown flags and missing values are usage errors.
func ParseFlags(allowed []string, args []string) (Flags, []string, error) {
	flags := Flags{}
	var positional []string
	allow := map[string]bool{}
	for _, a := range allowed {
		allow[a] = true
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			positional = append(positional, arg)
			continue
		}
		name := strings.TrimPrefix(arg, "--")
		var val string
		if eq := strings.Index(name, "="); eq >= 0 {
			name, val = name[:eq], name[eq+1:]
			if !allow[name] {
				return nil, nil, usageErr("unknown flag --%s", name)
			}
		} else {
			if !allow[name] {
				return nil, nil, usageErr("unknown flag --%s", name)
			}
			if i+1 >= len(args) {
				return nil, nil, usageErr("flag --%s requires a value", name)
			}
			i++
			val = args[i]
		}
		flags[name] = val
	}
	return flags, positional, nil
}

// Need returns a usage error unless every named flag has a non-empty value.
func (f Flags) Need(names ...string) error {
	for _, n := range names {
		if f[n] == "" {
			return usageErr("missing required flag --%s", n)
		}
	}
	return nil
}
