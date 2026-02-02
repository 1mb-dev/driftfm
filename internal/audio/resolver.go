package audio

import (
	"fmt"
	"path"
	"strings"
)

// Resolver resolves logical file paths to playable URLs
type Resolver interface {
	ResolveURL(filePath string) (string, error)
}

// NewResolver creates a local file resolver for the given base path
func NewResolver(basePath string) Resolver {
	// Normalize path to avoid double slashes
	return &LocalResolver{BasePath: "/" + strings.Trim(basePath, "/")}
}

// sanitizePath cleans a file path to prevent traversal attacks
func sanitizePath(filePath string) string {
	// Clean the path and remove any traversal attempts
	cleaned := path.Clean(filePath)
	// Remove leading slashes and parent references
	cleaned = strings.TrimPrefix(cleaned, "/")
	for strings.HasPrefix(cleaned, "../") {
		cleaned = strings.TrimPrefix(cleaned, "../")
	}
	return cleaned
}

// LocalResolver returns local file server paths
type LocalResolver struct {
	BasePath string // e.g., "/audio"
}

// ResolveURL returns the local path for a track
func (r *LocalResolver) ResolveURL(filePath string) (string, error) {
	safe := sanitizePath(filePath)
	return fmt.Sprintf("%s/%s", r.BasePath, safe), nil
}
