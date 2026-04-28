package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/joho/godotenv"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		fmt.Fprintf(os.Stderr, "no .env file, using system env\n")
	}

	var prompt string
	var model string

	flag.StringVar(&prompt, "p", "", "Prompt to send to LLM")
	flag.StringVar(&model, "m", os.Getenv("MODEL_NAME"), "Model name")
	flag.Parse()

	if prompt == "" {
		log.Fatal("Prompt must not be empty")
	}
	if model == "" {
		log.Fatal("Model must not be empty")
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	baseUrl := os.Getenv("OPENROUTER_BASE_URL")

	if baseUrl == "" {
		baseUrl = "https://openrouter.ai/api/v1"
	}

	if apiKey == "" {
		log.Fatal("Env variable OPENROUTER_API_KEY not found")
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseUrl
	client := openai.NewClientWithConfig(config)

	messages := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: `Ты - автономный агент-разработчик. 
Сначала попытайся собрать информацию по проекту, а не галлюционируй. Это ОБЯЗАТЕЛЬНО.
Для выполнения ЛЮБЫХ действий с файлами (чтение, запись, выполнение команд) ты ОБЯЗАН использовать предоставленные инструменты.
`,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	fmt.Fprintln(os.Stderr, "Logs from your program will appear here!")

	inCycle := true
	for inCycle {
		request := openai.ChatCompletionRequest{
			Model:    model,
			Messages: messages,
			Tools: []openai.Tool{
				{
					Type: openai.ToolTypeFunction,
					Function: &openai.FunctionDefinition{
						Name:        "Read",
						Description: "Read and return the contents of a file",
						Parameters: jsonschema.Definition{
							Type: jsonschema.Object,
							Properties: map[string]jsonschema.Definition{
								"file_path": {
									Type:        jsonschema.String,
									Description: "The path to the file to read",
								},
							},
							Required: []string{"file_path"},
						},
					},
				},
				{
					Type: openai.ToolTypeFunction,
					Function: &openai.FunctionDefinition{
						Name:        "Write",
						Description: "Write content to a file",
						Parameters: jsonschema.Definition{
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
						},
					},
				},
				{
					Type: openai.ToolTypeFunction,
					Function: &openai.FunctionDefinition{
						Name:        "Bash",
						Description: "Execute a shell command",
						Parameters: jsonschema.Definition{
							Type: jsonschema.Object,
							Properties: map[string]jsonschema.Definition{
								"command": {
									Type:        jsonschema.String,
									Description: "The command to execute",
								},
							},
							Required: []string{"command"},
						},
					},
				},
			},
		}

		resp, err := client.CreateChatCompletion(context.Background(), request)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if len(resp.Choices) == 0 {
			log.Fatal("No choices in response")
		}

		data, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Fprintln(os.Stderr, string(data))

		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) > 0 {
			// Append assistant's message with tool calls to history
			messages = append(messages, msg)

			for _, toolCall := range msg.ToolCalls {
				switch toolCall.Function.Name {
				case "Read":
					type props struct {
						FilePath string `json:"file_path"`
					}
					var prop props
					json.Unmarshal([]byte(toolCall.Function.Arguments), &prop)

					fmt.Fprintln(os.Stderr, prop.FilePath)

					if contents, errF := os.ReadFile(prop.FilePath); errF != nil {
						errMsj := fmt.Sprintf("got error while reading file: %v", errF)
						fmt.Fprintln(os.Stderr, errMsj)
						messages = append(messages, openai.ChatCompletionMessage{
							Role:       openai.ChatMessageRoleTool,
							Content:    errMsj,
							ToolCallID: toolCall.ID,
						})
					} else {
						messages = append(messages, openai.ChatCompletionMessage{
							Role:       openai.ChatMessageRoleTool,
							Content:    string(contents),
							ToolCallID: toolCall.ID,
						})
					}

				case "Write":
					type props struct {
						FilePath string `json:"file_path"`
						Content  string `json:"content"`
					}
					var prop props
					json.Unmarshal([]byte(toolCall.Function.Arguments), &prop)

					fmt.Fprintln(os.Stderr, prop.FilePath)
					fmt.Fprintln(os.Stderr, prop.Content)

					if errW := os.WriteFile(prop.FilePath, []byte(prop.Content), 0644); errW != nil {
						errMsj := fmt.Sprintf("failed to write file: %v", errW)
						fmt.Fprintln(os.Stderr, errMsj)
						messages = append(messages, openai.ChatCompletionMessage{
							Role:       openai.ChatMessageRoleTool,
							Content:    errMsj,
							ToolCallID: toolCall.ID,
						})
					} else {
						successMsg := "File written successfully"
						fmt.Fprintln(os.Stderr, successMsg)
						messages = append(messages, openai.ChatCompletionMessage{
							Role:       openai.ChatMessageRoleTool,
							Content:    successMsg,
							ToolCallID: toolCall.ID,
						})
					}

				case "Bash":
					type props struct {
						Command string `json:"command"`
					}
					var prop props
					json.Unmarshal([]byte(toolCall.Function.Arguments), &prop)

					fmt.Fprintln(os.Stderr, prop.Command)

					if prop.Command != "" {
						fmt.Fprintln(os.Stderr, "Executing command:", prop.Command)
						cmd := exec.Command("sh", "-c", prop.Command)
						output, errE := cmd.CombinedOutput()

						var responseContent string
						if errE != nil {
							responseContent = fmt.Sprintf("Command failed: %v\nOutput: %s", errE, string(output))
							fmt.Fprintln(os.Stderr, responseContent)
						} else {
							responseContent = fmt.Sprintf("Command executed successfully:\n%s", string(output))
							fmt.Fprintln(os.Stderr, "Command success")
						}

						messages = append(messages, openai.ChatCompletionMessage{
							Role:       openai.ChatMessageRoleTool,
							Content:    responseContent,
							ToolCallID: toolCall.ID,
						})
					}
				}
			}
		} else {
			fmt.Print(msg.Content)
			inCycle = false
		}
	}
}
