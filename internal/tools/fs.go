package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sashabaranov/go-openai"
)

func (r *Registry) registerRead() {
	r.registerTool(openai.FunctionDefinition{
		Name:        "Read",
		Description: "Read and return the contents of a file",
		Parameters:  filePathSchema(),
	}, func(rawArgs string) (string, error) {
		type props struct {
			FilePath string `json:"file_path"`
		}
		var p props
		if err := mustUnmarshal(rawArgs, &p); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		path, err := r.safePath(p.FilePath)
		if err != nil {
			return "", err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("got error while reading file: %w", err)
		}
		return string(b), nil
	})
}

func (r *Registry) registerWrite() {
	r.registerTool(openai.FunctionDefinition{
		Name:        "Write",
		Description: "Write content to a file",
		Parameters:  fileWriteSchema(),
	}, func(rawArgs string) (string, error) {
		type props struct {
			FilePath string `json:"file_path"`
			Content  string `json:"content"`
		}
		var p props
		if err := mustUnmarshal(rawArgs, &p); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		path, err := r.safePath(p.FilePath)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", fmt.Errorf("failed to create parent directories: %w", err)
		}
		if err := os.WriteFile(path, []byte(p.Content), 0o644); err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}
		return "File written successfully", nil
	})
}

func (r *Registry) safePath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("file_path must not be empty")
	}

	// Treat relative paths as rooted in workspaceRoot.
	if !filepath.IsAbs(p) {
		p = filepath.Join(r.workspaceRoot, p)
	}

	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	abs = filepath.Clean(abs)

	root := filepath.Clean(r.workspaceRoot)
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path is outside workspace root: %s", abs)
	}

	return abs, nil
}
