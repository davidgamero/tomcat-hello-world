package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
)

func main() {
	// Get environment variables
	apiKey := os.Getenv("AZURE_OPENAI_KEY")
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")

	if apiKey == "" || endpoint == "" {
		fmt.Println("Error: AZURE_OPENAI_KEY and AZURE_OPENAI_ENDPOINT environment variables must be set")
		os.Exit(1)
	}

	// Create a client with KeyCredential
	keyCredential := azcore.NewKeyCredential(apiKey)
	client, err := azopenai.NewClientWithKeyCredential(endpoint, keyCredential, nil)
	if err != nil {
		fmt.Printf("Error creating Azure OpenAI client: %v\n", err)
		os.Exit(1)
	}

	deploymentID := "o3-mini"

	resp, err := client.GetChatCompletions(
		context.Background(),
		azopenai.ChatCompletionsOptions{
			DeploymentName: to.Ptr(deploymentID),
			Messages: []azopenai.ChatRequestMessageClassification{
				&azopenai.ChatRequestUserMessage{
					Content: azopenai.NewChatRequestUserMessageContent("Hello Azure OpenAI! Tell me this is working in one short sentence."),
				},
			},
		},
		nil,
	)
	if err != nil {
		fmt.Printf("Error getting chat completions: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Azure OpenAI Test:")
	if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != nil {
		fmt.Printf("Response: %s\n", *resp.Choices[0].Message.Content)
	}
}
