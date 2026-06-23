package infra

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// GetPromptTemplate 从 internal/prompts/{name}.md 读取 Agent 的系统提示词。
// 这样提示词可以独立维护，不需要写死在 Go 代码里。
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
