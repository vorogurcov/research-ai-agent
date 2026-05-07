package tools

import "github.com/sashabaranov/go-openai/jsonschema"

// shared parameter schemas
func filePathSchema() jsonschema.Definition {
	return jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"file_path": {
				Type:        jsonschema.String,
				Description: "The path to the file to read",
			},
		},
		Required: []string{"file_path"},
	}
}

// shared parameter schemas
func fileWriteSchema() jsonschema.Definition {
	return jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"task_name": {
				Type:        jsonschema.String,
				Description: "A short name for the current task (used to group all writes under writes/{task_name}_.../).",
			},
			"file_path": {
				Type:        jsonschema.String,
				Description: "The file name to write (directories will be ignored; output is always placed under writes/...).",
			},
			"content": {
				Type:        jsonschema.String,
				Description: "The content to write to the file",
			},
		},
		Required: []string{"task_name", "file_path", "content"},
	}
}

func writesLsSchema() jsonschema.Definition {
	return jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"path": {
				Type:        jsonschema.String,
				Description: "Optional subpath within writes/ to list (relative). Empty means list writes/ root.",
			},
		},
	}
}

func registerBashSchema() jsonschema.Definition {
	return jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"command": {
				Type:        jsonschema.String,
				Description: "The command to execute",
			},
		},
		Required: []string{"command"},
	}
}
