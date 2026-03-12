package scaffold

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ProcessTemplate replaces {{key}} placeholders in tmplContent with
// the corresponding values from the values map.
// Only simple {{key}} syntax is supported (no Go template features).
func ProcessTemplate(tmplContent string, values map[string]string) (string, error) {
	result := tmplContent
	for k, v := range values {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result, nil
}

// ProcessTemplatePath renders template variables in a file path string.
func ProcessTemplatePath(path string, values map[string]string) (string, error) {
	if !strings.Contains(path, "{{") {
		return path, nil
	}
	return ProcessTemplate(path, values)
}

// CollectOptionValues collects option values from provided args, defaults, or interactive input.
// When useDefaults is true, non-required options with defaults are auto-applied without prompting.
// All options (required and non-required) are prompted interactively unless overridden
// by provided values or auto-applied defaults.
func CollectOptionValues(options []Option, provided map[string]string,
	reader io.Reader, writer io.Writer, useDefaults bool) (map[string]string, error) {

	if len(options) == 0 {
		return map[string]string{}, nil
	}

	values := make(map[string]string, len(options))
	scanner := bufio.NewScanner(reader)

	for _, opt := range options {
		// Check if value was provided via --v flags
		if val, ok := provided[opt.Name]; ok {
			values[opt.Name] = val
			continue
		}

		// Auto-apply defaults for non-required options when --default is set
		if useDefaults && !opt.Required && opt.Default != "" {
			values[opt.Name] = opt.Default
			continue
		}

		// Interactive prompt for all remaining options
		if writer != nil {
			if opt.Default != "" {
				fmt.Fprintf(writer, "? %s (%s) (%s): ", opt.Description, opt.Name, opt.Default)
			} else {
				fmt.Fprintf(writer, "? %s (%s): ", opt.Description, opt.Name)
			}
		}

		if scanner.Scan() {
			val := strings.TrimSpace(scanner.Text())
			if val == "" {
				// Empty enter: apply default if available
				if opt.Default != "" {
					values[opt.Name] = opt.Default
				} else if opt.Required {
					return nil, fmt.Errorf("required option %q cannot be empty", opt.Name)
				} else {
					values[opt.Name] = ""
				}
			} else {
				values[opt.Name] = val
			}
		} else {
			if opt.Default != "" {
				values[opt.Name] = opt.Default
			} else if opt.Required {
				return nil, fmt.Errorf("required option %q: no input received", opt.Name)
			} else {
				values[opt.Name] = ""
			}
		}
	}

	return values, nil
}

// ParseOptionOverrides parses --v key=value arguments into a map.
func ParseOptionOverrides(args []string) (map[string]string, error) {
	result := make(map[string]string, len(args))
	for _, arg := range args {
		idx := strings.Index(arg, "=")
		if idx < 0 {
			return nil, fmt.Errorf("invalid option override %q: expected key=value format", arg)
		}
		key := arg[:idx]
		if key == "" {
			return nil, fmt.Errorf("invalid option override %q: key cannot be empty", arg)
		}
		value := arg[idx+1:]
		result[key] = value
	}
	return result, nil
}
