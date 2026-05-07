package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sashabaranov/go-openai"
)

func (r *Registry) registerRead() {
	r.registerTool(openai.FunctionDefinition{
		Name:        "Read",
		Description: "Read and return the contents of a file. For safety, this tool can only read files under the writes/ directory. Use the relative path returned by Write (e.g. task_YYYYMMDD_HHMMSS_001/output.json). Absolute paths and .. are rejected.",
		Parameters:  filePathSchema(),
	}, func(rawArgs string) (string, error) {
		type props struct {
			FilePath string `json:"file_path"`
		}
		var p props
		if err := mustUnmarshal(rawArgs, &p); err != nil {
			r.writeError("tools.Read.mustUnmarshal", err)
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		path, err := r.safeWritesPath(p.FilePath)
		if err != nil {
			r.writeError("tools.Read.safeWritesPath", err)
			return "", err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			r.writeError("tools.Read.os.ReadFile", err)
			return "", fmt.Errorf("got error while reading file: %w", err)
		}
		return string(b), nil
	})
}

func (r *Registry) registerLs() {
	r.registerTool(openai.FunctionDefinition{
		Name:        "Ls",
		Description: "List files and directories under writes/. Returns JSON array of entries with relative_path and type ('file'|'dir').",
		Parameters:  writesLsSchema(),
	}, func(rawArgs string) (string, error) {
		type props struct {
			Path string `json:"path"`
		}
		var p props
		if strings.TrimSpace(rawArgs) != "" {
			if err := mustUnmarshal(rawArgs, &p); err != nil {
				r.writeError("tools.Ls.mustUnmarshal", err)
				return "", fmt.Errorf("invalid arguments: %w", err)
			}
		}

		dirPath, relBase, err := r.safeWritesDirPath(p.Path)
		if err != nil {
			r.writeError("tools.Ls.safeWritesDirPath", err)
			return "", err
		}

		entries, err := os.ReadDir(dirPath)
		if err != nil {
			r.writeError("tools.Ls.os.ReadDir", err)
			return "", fmt.Errorf("failed to list directory: %w", err)
		}

		type outEntry struct {
			RelativePath string `json:"relative_path"`
			Type         string `json:"type"`
		}

		out := make([]outEntry, 0, len(entries))
		for _, e := range entries {
			t := "file"
			if e.IsDir() {
				t = "dir"
			}
			rp := filepath.ToSlash(filepath.Join(relBase, e.Name()))
			rp = strings.TrimPrefix(rp, "/")
			out = append(out, outEntry{
				RelativePath: rp,
				Type:         t,
			})
		}

		b, err := json.Marshal(out)
		if err != nil {
			r.writeError("tools.Ls.json.Marshal", err)
			return "", fmt.Errorf("failed to encode output: %w", err)
		}
		return string(b), nil
	})
}

func (r *Registry) safeWritesPath(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		err := fmt.Errorf("file_path must not be empty")
		r.writeError("tools.safeWritesPath", err)
		return "", err
	}

	// Disallow absolute paths: Read is strictly sandboxed to writes/.
	if filepath.IsAbs(p) {
		err := fmt.Errorf("absolute paths are not allowed for Read: %s", p)
		r.writeError("tools.safeWritesPath.absolute", err)
		return "", err
	}

	// Root for allowed reads: <workspaceRoot>/writes
	root := filepath.Join(r.workspaceRoot, "writes")
	root = filepath.Clean(root)

	// Make path relative to writes/ and normalize.
	joined := filepath.Join(root, p)
	abs, err := filepath.Abs(joined)
	if err != nil {
		r.writeError("tools.safeWritesPath.filepath.Abs", err)
		return "", fmt.Errorf("invalid path: %w", err)
	}
	abs = filepath.Clean(abs)

	rel, err := filepath.Rel(root, abs)
	if err != nil {
		r.writeError("tools.safeWritesPath.filepath.Rel", err)
		return "", fmt.Errorf("invalid path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		err := fmt.Errorf("path is outside writes directory: %s", abs)
		r.writeError("tools.safeWritesPath.outsideWrites", err)
		return "", err
	}

	return abs, nil
}

func (r *Registry) safeWritesDirPath(p string) (absDir string, relBase string, err error) {
	// Root for allowed reads: <workspaceRoot>/writes
	root := filepath.Join(r.workspaceRoot, "writes")
	root = filepath.Clean(root)

	sub := strings.TrimSpace(p)
	if sub == "" {
		return root, "", nil
	}
	if filepath.IsAbs(sub) {
		return "", "", fmt.Errorf("absolute paths are not allowed for Ls: %s", sub)
	}

	joined := filepath.Join(root, sub)
	abs, err := filepath.Abs(joined)
	if err != nil {
		return "", "", fmt.Errorf("invalid path: %w", err)
	}
	abs = filepath.Clean(abs)

	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", "", fmt.Errorf("invalid path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("path is outside writes directory: %s", abs)
	}

	return abs, filepath.ToSlash(rel), nil
}

func (r *Registry) registerWrite() {
	r.registerTool(openai.FunctionDefinition{
		Name:        "Write",
		Description: "Write content to a file. Output is always written under writes/{task_name}_{datetime}_{index}/ to avoid creating files across the repo.",
		Parameters:  fileWriteSchema(),
	}, func(rawArgs string) (string, error) {
		type props struct {
			TaskName string `json:"task_name"`
			FilePath string `json:"file_path"`
			Content  string `json:"content"`
		}
		var p props
		if err := mustUnmarshal(rawArgs, &p); err != nil {
			r.writeError("tools.Write.mustUnmarshal", err)
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		task := sanitizeTaskName(p.TaskName)
		if task == "" {
			task = "task"
		}

		filename := filepath.Base(strings.TrimSpace(p.FilePath))
		filename = strings.TrimSpace(filename)
		if filename == "" || filename == "." || filename == string(filepath.Separator) {
			filename = "output.txt"
		}

		dir := filepath.Join("writes", r.nextWriteDir(task))
		path, err := r.safePath(filepath.Join(dir, filename))
		if err != nil {
			r.writeError("tools.Write.safePath", err)
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			r.writeError("tools.Write.os.MkdirAll", err)
			return "", fmt.Errorf("failed to create parent directories: %w", err)
		}
		if err := os.WriteFile(path, []byte(p.Content), 0o644); err != nil {
			r.writeError("tools.Write.os.WriteFile", err)
			return "", fmt.Errorf("failed to write file: %w", err)
		}
		return "File written successfully: " + filepath.ToSlash(filepath.Join(dir, filename)), nil
	})
}

var taskNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func sanitizeTaskName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, " ", "_")
	s = taskNameSanitizer.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_-")
	return s
}

func (r *Registry) safePath(p string) (string, error) {
	if p == "" {
		err := fmt.Errorf("file_path must not be empty")
		r.writeError("tools.safePath", err)
		return "", err
	}

	// Treat relative paths as rooted in workspaceRoot.
	if !filepath.IsAbs(p) {
		p = filepath.Join(r.workspaceRoot, p)
	}

	abs, err := filepath.Abs(p)
	if err != nil {
		r.writeError("tools.safePath.filepath.Abs", err)
		return "", fmt.Errorf("invalid path: %w", err)
	}
	abs = filepath.Clean(abs)

	root := filepath.Clean(r.workspaceRoot)
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		r.writeError("tools.safePath.filepath.Rel", err)
		return "", fmt.Errorf("invalid path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		err := fmt.Errorf("path is outside workspace root: %s", abs)
		r.writeError("tools.safePath.outsideWorkspace", err)
		return "", err
	}

	return abs, nil
}
