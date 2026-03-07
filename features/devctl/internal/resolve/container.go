package resolve

import (
	"regexp"
	"strings"
)

var invalidChars = regexp.MustCompile(`[^a-z0-9-]`)

// sanitize converts a string to a valid container name component.
func sanitize(s string) string {
	return invalidChars.ReplaceAllString(strings.ToLower(s), "-")
}

// ContainerName returns "<project>-<feature>" with invalid chars replaced.
func ContainerName(project, feature string) string {
	return sanitize(project) + "-" + sanitize(feature)
}

// ImageName returns "<project>-dev-<feature>" with invalid chars replaced.
func ImageName(project, feature string) string {
	return sanitize(project) + "-dev-" + sanitize(feature)
}
