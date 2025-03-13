package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// Message represents a single message in the OpenAI chat format
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Request represents the request payload for the Azure OpenAI API
type Request struct {
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

// Choice represents a single response choice from the API
type Choice struct {
	Message Message `json:"message"`
}

// Response represents the response from the Azure OpenAI API
type Response struct {
	Choices []Choice `json:"choices"`
}

// AzureOpenAIClient handles communication with Azure OpenAI API
type AzureOpenAIClient struct {
	APIKey         string
	Endpoint       string
	DeploymentName string
	APIVersion     string
}

// NewClient creates a new Azure OpenAI client with the given credentials
func NewClient(apiKey, endpoint string) *AzureOpenAIClient {
	return &AzureOpenAIClient{
		APIKey:         apiKey,
		Endpoint:       endpoint,
		DeploymentName: "o3-mini",
		APIVersion:     "2025-01-31",
	}
}

// SetDeployment sets the deployment name
func (c *AzureOpenAIClient) SetDeployment(deploymentName string) {
	c.DeploymentName = deploymentName
}

// SetAPIVersion sets the API version
func (c *AzureOpenAIClient) SetAPIVersion(apiVersion string) {
	c.APIVersion = apiVersion
}

// CompleteChat sends a chat completion request to Azure OpenAI
func (c *AzureOpenAIClient) CompleteChat(systemPrompt, userPrompt string, maxTokens int) (string, error) {
	// Create request payload
	req := Request{
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens: maxTokens,
	}

	// Construct API URL
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		c.Endpoint, c.DeploymentName, c.APIVersion)

	// Convert payload to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("error creating JSON request: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating HTTP request: %v", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", c.APIKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	// Check status code
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("error: received status code %d, response: %s",
			resp.StatusCode, string(body))
	}

	// Parse response
	var result Response
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("error parsing response: %v, body: %s", err, string(body))
	}

	// Get AI message
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("error: no choices in response, body: %s", string(body))
	}

	// Return the content of the message
	return result.Choices[0].Message.Content, nil
}

// SaveResponseToFile saves the given content to a file
func SaveResponseToFile(content, filename string) error {
	return ioutil.WriteFile(filename, []byte(content), 0644)
}

func main() {
	// Get configuration from environment variables
	apiKey := os.Getenv("AZURE_OPENAI_KEY")
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	fmt.Println("Azure OpenAI Endpoint:", endpoint)

	if apiKey == "" || endpoint == "" {
		fmt.Println("Error: AZURE_OPENAI_KEY or AZURE_OPENAI_ENDPOINT environment variables not set")
		os.Exit(1)
	}

	// Create client
	client := NewClient(apiKey, endpoint)

	// Example usage
	systemPrompt := "You are an AI assistant."
	userPrompt := "Say hello world!"
	maxTokens := 100

	fmt.Println("Sending request to Azure OpenAI...")

	response, err := client.CompleteChat(systemPrompt, userPrompt, maxTokens)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Print response
	fmt.Println("\nAzure OpenAI Response:")
	fmt.Printf("AI: %s\n", response)

	// Save response to file
	outputFile := "azure_openai_response.txt"
	err = SaveResponseToFile(response, outputFile)
	if err != nil {
		fmt.Printf("Error saving response to file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Response saved to %s\n", outputFile)
}
