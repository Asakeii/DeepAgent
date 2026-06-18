package infra

// TODO: Load prompt templates from internal/prompts.
import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// GetPromptTemplate loads a prompt template from internal/prompts/{name}.md.
func GetPromptTemplate(ctx context.Context, name string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	path := filepath.Join(wd, "internal", "prompts", fmt.Sprintf("%s.md", name))
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read prompt %s: %w", path, err)
	}

	return string(data), nil
}
