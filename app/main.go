package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
)

func main() {
	var prompt string
	var model string
	flag.StringVar(&prompt, "p", "", "Prompt to send to LLM")
	flag.StringVar(&model, "m", "anthropic/claude-haiku-4.5", "Model name")
	flag.Parse()

	if prompt == "" {
		panic("Prompt must not be empty")
	}

	if err := godotenv.Load(".env"); err != nil {
		fmt.Fprintf(os.Stderr, "no .env file, using system env")
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	baseUrl := os.Getenv("OPENROUTER_BASE_URL")
	if baseUrl == "" {
		baseUrl = "https://openrouter.ai/api/v1"
	}

	if apiKey == "" {
		panic("Env variable OPENROUTER_API_KEY not found")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseUrl))

	messages := make([]openai.ChatCompletionMessageParamUnion, 0)
	messages = append(messages, openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfString: openai.String(prompt),
			},
		},
	})
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Fprintln(os.Stderr, "Logs from your program will appear here!")
	inCycle := true
	for inCycle {
		request := openai.ChatCompletionNewParams{
			Model:    model,
			Messages: messages,
			Tools: []openai.ChatCompletionToolUnionParam{
				{
					OfFunction: &openai.ChatCompletionFunctionToolParam{
						Function: openai.FunctionDefinitionParam{
							Name:        "Read",
							Description: param.NewOpt("Read and return the contents of a file"),
							Parameters: openai.FunctionParameters{
								"type": "object",
								"properties": map[string]any{
									"file_path": map[string]any{
										"type":        "string",
										"description": "The path to the file to read",
									},
								},
								"required": []string{"file_path"},
							},
						},
						Type: "function",
					},
				},
				{
					OfFunction: &openai.ChatCompletionFunctionToolParam{
						Function: openai.FunctionDefinitionParam{
							Name:        "Write",
							Description: param.NewOpt("Write content to a file"),
							Parameters: openai.FunctionParameters{
								"type": "object",
								"properties": map[string]any{
									"file_path": map[string]any{
										"type":        "string",
										"description": "The path of the file to write to",
									},
									"content": map[string]any{
										"type":        "string",
										"description": "The content to write to the file",
									},
								},
								"required": []string{"file_path", "content"},
							},
						},
						Type: "function",
					},
				},
				{
					OfFunction: &openai.ChatCompletionFunctionToolParam{
						Function: openai.FunctionDefinitionParam{
							Name:        "Bash",
							Description: param.NewOpt("Execute a shell command"),
							Parameters: openai.FunctionParameters{
								"type": "object",
								"properties": map[string]any{
									"command": map[string]any{
										"type":        "string",
										"description": "The command to execute",
									},
								},
								"required": []string{"command"},
							},
						},
						Type: "function",
					},
				},
			},
		}
		resp, err := client.Chat.Completions.New(context.Background(), request)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if len(resp.Choices) == 0 {
			panic("No choices in response")
		}
		data, err := json.MarshalIndent(resp, "", "  ")
		fmt.Fprintln(os.Stderr, string(data))

		if len(resp.Choices[0].Message.ToolCalls) > 0 {
			switch {
			case resp.Choices[0].Message.ToolCalls[0].Function.Name == "Read":
				toolCall := resp.Choices[0].Message.ToolCalls[0]
				args := toolCall.Function.Arguments

				assistantMsg := resp.Choices[0].Message
				messages = append(messages, openai.ChatCompletionMessageParamUnion{
					OfAssistant: &openai.ChatCompletionAssistantMessageParam{
						Content: openai.ChatCompletionAssistantMessageParamContentUnion{
							OfString: param.NewOpt[string](assistantMsg.Content),
						},
						ToolCalls: []openai.ChatCompletionMessageToolCallUnionParam{
							{
								OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
									ID: toolCall.ID,
									Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
										Arguments: args,
										Name:      "Read",
									},
									Type: "function",
								},
							},
						},
						Role: "assistant",
					},
				})

				type props struct {
					FilePath string `json:"file_path"`
				}
				var prop props
				json.Unmarshal([]byte(args), &prop)

				fmt.Fprintln(os.Stderr, prop.FilePath)

				if contents, errF := os.ReadFile(prop.FilePath); errF != nil {
					fmt.Fprintln(os.Stderr, fmt.Errorf("got error while reading file: %v", errF))
				} else {
					messages = append(messages, openai.ChatCompletionMessageParamUnion{
						OfTool: &openai.ChatCompletionToolMessageParam{
							Content: openai.ChatCompletionToolMessageParamContentUnion{
								OfString: param.NewOpt[string](string(contents)),
							},
							ToolCallID: toolCall.ID,
							Role:       "tool",
						},
					})
				}
			case resp.Choices[0].Message.ToolCalls[0].Function.Name == "Write":
				toolCall := resp.Choices[0].Message.ToolCalls[0]
				args := toolCall.Function.Arguments

				assistantMsg := resp.Choices[0].Message
				messages = append(messages, openai.ChatCompletionMessageParamUnion{
					OfAssistant: &openai.ChatCompletionAssistantMessageParam{
						Content: openai.ChatCompletionAssistantMessageParamContentUnion{
							OfString: param.NewOpt[string](assistantMsg.Content),
						},
						ToolCalls: []openai.ChatCompletionMessageToolCallUnionParam{
							{
								OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
									ID: toolCall.ID,
									Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
										Arguments: args,
										Name:      "Write",
									},
									Type: "function",
								},
							},
						},
						Role: "assistant",
					},
				})

				type props struct {
					FilePath string `json:"file_path"`
					Content  string `json:"content"`
				}
				var prop props
				json.Unmarshal([]byte(args), &prop)

				fmt.Fprintln(os.Stderr, prop.FilePath)
				fmt.Fprintln(os.Stderr, prop.Content)

				if errW := os.WriteFile(prop.FilePath, []byte(prop.Content), 0644); errW != nil {
					errMsj := fmt.Sprintf("failed to write file: %v", errW)
					fmt.Fprintln(os.Stderr, errMsj)
				} else {
					successMsg := "File written successfully"
					fmt.Fprintln(os.Stderr, successMsg)

					messages = append(messages, openai.ChatCompletionMessageParamUnion{
						OfTool: &openai.ChatCompletionToolMessageParam{
							Content:    openai.ChatCompletionToolMessageParamContentUnion{OfString: param.NewOpt[string](successMsg)},
							ToolCallID: toolCall.ID,
							Role:       "tool",
						},
					})
				}
			case resp.Choices[0].Message.ToolCalls[0].Function.Name == "Bash":
				toolCall := resp.Choices[0].Message.ToolCalls[0]
				args := toolCall.Function.Arguments

				assistantMsg := resp.Choices[0].Message
				messages = append(messages, openai.ChatCompletionMessageParamUnion{
					OfAssistant: &openai.ChatCompletionAssistantMessageParam{
						Content: openai.ChatCompletionAssistantMessageParamContentUnion{
							OfString: param.NewOpt[string](assistantMsg.Content),
						},
						ToolCalls: []openai.ChatCompletionMessageToolCallUnionParam{
							{
								OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
									ID: toolCall.ID,
									Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
										Arguments: args,
										Name:      "Bash",
									},
									Type: "function",
								},
							},
						},
						Role: "assistant",
					},
				})

				type props struct {
					Command string `json:"command"`
				}
				var prop props
				json.Unmarshal([]byte(args), &prop)

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

					messages = append(messages, openai.ChatCompletionMessageParamUnion{
						OfTool: &openai.ChatCompletionToolMessageParam{
							Content:    openai.ChatCompletionToolMessageParamContentUnion{OfString: param.NewOpt[string](responseContent)},
							ToolCallID: toolCall.ID,
							Role:       "tool",
						},
					})
				}
			}

		} else {
			fmt.Print(resp.Choices[0].Message.Content)
			inCycle = false
		}
	}
}
