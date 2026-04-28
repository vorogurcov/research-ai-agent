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
			"file_path": {
				Type:        jsonschema.String,
				Description: "The path of the file to write to",
			},
			"content": {
				Type:        jsonschema.String,
				Description: "The content to write to the file",
			},
		},
		Required: []string{"file_path", "content"},
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
