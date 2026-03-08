package scaffold

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"
)

// ProcessTemplate renders a Go template string with the given values.
func ProcessTemplate(tmplContent string, values map[string]string) (string, error) {
	t, err := template.New("scaffold").Option("missingkey=zero").Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, values); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// ProcessTemplatePath renders template variables in a file path string.
func ProcessTemplatePath(path string, values map[string]string) (string, error) {
	if !strings.Contains(path, "{{") {
		return path, nil
	}
	return ProcessTemplate(path, values)
}

// CollectOptionValues collects option values from provided args, defaults, or interactive input.
// For required options not in provided, it prompts the user via reader/writer.
func CollectOptionValues(options []Option, provided map[string]string,
	reader io.Reader, writer io.Writer) (map[string]string, error) {

	if len(options) == 0 {
		return map[string]string{}, nil
	}

	values := make(map[string]string, len(options))
	scanner := bufio.NewScanner(reader)

	for _, opt := range options {
		// Check if value was provided via CLI flags
		if val, ok := provided[opt.Name]; ok {
			values[opt.Name] = val
			continue
		}

		// Check if there's a default value
		if opt.Default != "" && !opt.Required {
			values[opt.Name] = opt.Default
			continue
		}

		// Required and not provided: prompt interactively
		if opt.Required {
			if writer != nil {
				fmt.Fprintf(writer, "? %s (%s): ", opt.Name, opt.Description)
			}
			if scanner.Scan() {
				val := strings.TrimSpace(scanner.Text())
				if val == "" {
					return nil, fmt.Errorf("required option %q cannot be empty", opt.Name)
				}
				values[opt.Name] = val
			} else {
				return nil, fmt.Errorf("required option %q: no input received", opt.Name)
			}
		}
	}

	return values, nil
}
