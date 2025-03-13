package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
)

// DockerfileInput represents the input for Dockerfile analysis
type DockerfileInput struct {
	DockerfileContent string `json:"dockerfile_content"` // Plain text content of the Dockerfile
	ErrorMessages     string `json:"error_messages,omitempty"`
	repoFileTree      string `json:"repo_files,omitempty"`      // String representation of the file tree
	DockerfilePath    string `json:"dockerfile_path,omitempty"` // Path to the original Dockerfile
}

// DockerfileResult represents the analysis result
type DockerfileResult struct {
	FixedDockerfile string `json:"fixed_dockerfile"`
	Analysis        string `json:"analysis"`
}

func analyzeDockerfile(client *azopenai.Client, deploymentID string, input DockerfileInput) (*DockerfileResult, error) {
	// Create prompt for analyzing the Dockerfile
	promptText := fmt.Sprintf(`Analyze the following Dockerfile for errors and suggest fixes:
Dockerfile:
%s
`, input.DockerfileContent)

	// Add error information if provided and not empty
	if input.ErrorMessages != "" {
		promptText += fmt.Sprintf(`
Errors encountered when running this Dockerfile:
%s
`, input.ErrorMessages)
	} else {
		promptText += `
No error messages were provided. Please check for potential issues in the Dockerfile.
`
	}

	// Add repository file information if provided
	if input.repoFileTree != "" {
		promptText += fmt.Sprintf(`
Repository files structure:
%s
`, input.repoFileTree)
	}

	promptText += `
Please:
1. Identify any issues in the Dockerfile
2. Provide a fixed version of the Dockerfile
3. Explain what changes were made and why

Output the fixed Dockerfile between <<<DOCKERFILE>>> tags.`

	resp, err := client.GetChatCompletions(
		context.Background(),
		azopenai.ChatCompletionsOptions{
			DeploymentName: to.Ptr(deploymentID),
			Messages: []azopenai.ChatRequestMessageClassification{
				&azopenai.ChatRequestUserMessage{
					Content: azopenai.NewChatRequestUserMessageContent(promptText),
				},
			},
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != nil {
		content := *resp.Choices[0].Message.Content

		// Extract the fixed Dockerfile from between the tags
		// Use regex to find content between <<<DOCKERFILE>>> tags
		re := regexp.MustCompile(`<<<DOCKERFILE>>>([\s\S]*?)<<<DOCKERFILE>>>`)
		matches := re.FindStringSubmatch(content)

		fixedDockerfile := ""
		if len(matches) > 1 {
			// Found the dockerfile between tags
			fixedDockerfile = strings.TrimSpace(matches[1])
		} else {
			// If tags aren't found, try to extract the dockerfile content intelligently
			// Look for multi-line dockerfile content after FROM
			fromRe := regexp.MustCompile(`(?m)^FROM[\s\S]*?$`)
			if fromMatches := fromRe.FindString(content); fromMatches != "" {
				// Simple heuristic: Consider everything from the first FROM as the dockerfile
				fixedDockerfile = fromMatches
			} else {
				// Fallback: use the entire content (not ideal but better than nothing)
				fixedDockerfile = content
			}
		}

		return &DockerfileResult{
			FixedDockerfile: fixedDockerfile,
			Analysis:        content,
		}, nil
	}

	return nil, fmt.Errorf("no response from AI model")
}

// buildDockerfile attempts to build the Docker image and returns any error output
func buildDockerfile(dockerfilePath string) (bool, string) {
	// First check if docker is installed and available in PATH
	if _, err := exec.LookPath("docker"); err != nil {
		errorMsg := "Docker executable not found in PATH. Please install Docker or ensure it's available in your PATH."
		fmt.Println(errorMsg)
		return false, errorMsg
	}

	cmd := exec.Command("docker", "build", "-f", dockerfilePath, "-t", "test-image:latest", ".")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		fmt.Println("Docker build failed with error:", err)
		return false, outputStr
	}

	return true, outputStr
}

// iterateDockerfileBuild attempts to iteratively fix and build the Dockerfile
func iterateDockerfileBuild(client *azopenai.Client, deploymentID string, dockerfilePath string, fileStructurePath string, maxIterations int) error {
	fmt.Printf("Starting Dockerfile build iteration process for: %s\n", dockerfilePath)

	// Read the original Dockerfile
	dockerfileContent, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return fmt.Errorf("error reading Dockerfile: %v", err)
	}

	// Get repository structure
	repoStructure, err := os.ReadFile(fileStructurePath)
	if err != nil {
		return fmt.Errorf("error reading repository structure: %v", err)
	}

	currentDockerfile := string(dockerfileContent)

	for i := 0; i < maxIterations; i++ {
		fmt.Printf("\n=== Iteration %d of %d ===\n", i+1, maxIterations)

		// Try to build
		success, buildOutput := buildDockerfile(dockerfilePath)
		if success {
			fmt.Println("🎉 Docker build succeeded!")
			return nil
		}

		fmt.Println("Docker build failed. Using AI to fix issues...")

		// Prepare input for AI analysis
		input := DockerfileInput{
			DockerfileContent: currentDockerfile,
			ErrorMessages:     buildOutput,
			repoFileTree:      string(repoStructure),
			DockerfilePath:    dockerfilePath,
		}

		// Get AI to fix the Dockerfile
		result, err := analyzeDockerfile(client, deploymentID, input)
		if err != nil {
			return fmt.Errorf("error in AI analysis: %v", err)
		}

		// Update the Dockerfile
		currentDockerfile = result.FixedDockerfile
		fmt.Println("AI suggested fixes:")
		fmt.Println(result.Analysis)

		// Write the fixed Dockerfile
		if err := os.WriteFile(dockerfilePath, []byte(currentDockerfile), 0644); err != nil {
			return fmt.Errorf("error writing fixed Dockerfile: %v", err)
		}

		fmt.Printf("Updated Dockerfile written. Attempting build again...\n")
		time.Sleep(1 * time.Second) // Small delay for readability
	}

	return fmt.Errorf("failed to fix Dockerfile after %d iterations", maxIterations)
}

func main() {
	// Get environment variables
	apiKey := os.Getenv("AZURE_OPENAI_KEY")
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	deploymentID := "o3-mini"

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

	// Check command line arguments
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "iterate-dockerfile-build":
			maxIterations := 5
			dockerfilePath := "../../../Dockerfile"
			fileStructurePath := "repo_structure.txt" // Updated default extension

			// Allow custom dockerfile path
			if len(os.Args) > 2 {
				dockerfilePath = os.Args[2]
			}

			// Allow file structure JSON path
			if len(os.Args) > 3 {
				fileStructurePath = os.Args[3]
			}

			// Allow custom max iterations
			if len(os.Args) > 4 {
				fmt.Sscanf(os.Args[4], "%d", &maxIterations)
			}

			if err := iterateDockerfileBuild(client, deploymentID, dockerfilePath, fileStructurePath, maxIterations); err != nil {
				fmt.Printf("Error in dockerfile iteration process: %v\n", err)
				os.Exit(1)
			}

		default:
			// Default behavior - test Azure OpenAI
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
		return
	}

	// If no arguments provided, print usage
	fmt.Println("Usage:")
	fmt.Println("  go run azure_openai_hello.go                          - Test Azure OpenAI connection")
	fmt.Println("  go run azure_openai_hello.go iterate-dockerfile-build [dockerfile-path] [file-structure-path] [max-iterations] - Iteratively build and fix a Dockerfile")
}
