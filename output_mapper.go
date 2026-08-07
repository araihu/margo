package margo

import (
	"fmt"
	"path/filepath"
	"strings"
)

// OutputMapper maps one known source file to one output path. It performs no
// discovery, globbing, collision resolution, or filesystem writes.
type OutputMapper interface {
	Map(sourcePath string) (string, error)
}

// AdjacentMapper writes the HTML sibling of a source file.
type AdjacentMapper struct {
	Extension string
}

func (m AdjacentMapper) Map(sourcePath string) (string, error) {
	if err := validateSourcePath(sourcePath); err != nil {
		return "", err
	}
	return replaceExtension(filepath.Clean(sourcePath), m.Extension), nil
}

// PreserveMapper preserves a source file's path relative to SourceRoot under
// OutputDir.
type PreserveMapper struct {
	SourceRoot string
	OutputDir  string
	Extension  string
}

func (m PreserveMapper) Map(sourcePath string) (string, error) {
	if err := validateSourcePath(sourcePath); err != nil {
		return "", err
	}
	root, output, source, err := absoluteMappingPaths(m.SourceRoot, m.OutputDir, sourcePath)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, source)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("margo preserve mapper: source %q is outside source root %q", sourcePath, m.SourceRoot)
	}
	return replaceExtension(filepath.Join(output, relative), m.Extension), nil
}

// FlatMapper writes each known source file directly below OutputDir.
type FlatMapper struct {
	OutputDir string
	Extension string
}

func (m FlatMapper) Map(sourcePath string) (string, error) {
	if err := validateSourcePath(sourcePath); err != nil {
		return "", err
	}
	if strings.TrimSpace(m.OutputDir) == "" {
		return "", fmt.Errorf("margo flat mapper: output directory is required")
	}
	base := filepath.Base(filepath.Clean(sourcePath))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "", fmt.Errorf("margo flat mapper: invalid source path %q", sourcePath)
	}
	return replaceExtension(filepath.Join(filepath.Clean(m.OutputDir), base), m.Extension), nil
}

func absoluteMappingPaths(sourceRoot, outputDir, sourcePath string) (string, string, string, error) {
	if strings.TrimSpace(sourceRoot) == "" {
		return "", "", "", fmt.Errorf("margo preserve mapper: source root is required")
	}
	if strings.TrimSpace(outputDir) == "" {
		return "", "", "", fmt.Errorf("margo preserve mapper: output directory is required")
	}
	root, err := filepath.Abs(filepath.Clean(sourceRoot))
	if err != nil {
		return "", "", "", fmt.Errorf("margo preserve mapper: source root: %w", err)
	}
	output, err := filepath.Abs(filepath.Clean(outputDir))
	if err != nil {
		return "", "", "", fmt.Errorf("margo preserve mapper: output directory: %w", err)
	}
	source, err := filepath.Abs(filepath.Clean(sourcePath))
	if err != nil {
		return "", "", "", fmt.Errorf("margo preserve mapper: source: %w", err)
	}
	return root, output, source, nil
}

func validateSourcePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("margo output mapper: source path is required")
	}
	if strings.ContainsRune(path, '\x00') {
		return fmt.Errorf("margo output mapper: source path contains NUL")
	}
	return nil
}

func replaceExtension(path, extension string) string {
	extension = normalizeExtension(extension)
	ext := filepath.Ext(path)
	if ext == "" {
		return path + extension
	}
	return strings.TrimSuffix(path, ext) + extension
}

func normalizeExtension(extension string) string {
	if extension == "" {
		return ".html"
	}
	if !strings.HasPrefix(extension, ".") || strings.ContainsAny(extension, `/\\`) || extension == "." || extension == ".." {
		return ".html"
	}
	return extension
}

var _ OutputMapper = AdjacentMapper{}
var _ OutputMapper = PreserveMapper{}
var _ OutputMapper = FlatMapper{}
