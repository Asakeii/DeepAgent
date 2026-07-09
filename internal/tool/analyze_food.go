package tool

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/security"
	"deepAgent/internal/store"
	"deepAgent/internal/toolruntime"
)

const maxImageBytes = 20 << 20 // 20MB image size limit

// analyzeInput 是 analyze_food 工具的入参。
type analyzeInput struct {
	ThreadID  string `json:"thread_id" jsonschema:"required" jsonschema_description:"会话 thread_id"`
	ImagePath string `json:"image_path" jsonschema:"required" jsonschema_description:"图片路径（本地文件路径或 HTTP/HTTPS URL）"`
}

// foodItem 是模型返回的单个食物分析结果。
type foodItem struct {
	Name     string  `json:"name"`
	Weight   float64 `json:"weight_grams"`
	Calories float64 `json:"calories_kcal"`
}

// analyzeResult 是 analyze_food 工具的输出。
type analyzeResult struct {
	Foods         []foodItem `json:"foods"`
	TotalCalories float64    `json:"total_calories_kcal"`
	Summary       string     `json:"summary"`
}

const analyzeFoodPrompt = `请分析这张图片中的所有食物。对每种食物：
1. 识别名称
2. 估算份量（克）
3. 估算热量（千卡）

请严格按以下 JSON 格式输出，不要包含其他文字：
{
  "foods": [
    {"name": "食物名称", "weight_grams": 100, "calories_kcal": 200}
  ],
  "total_calories_kcal": 500,
  "summary": "一句话总结这顿饭的营养特点"
}`

// analyzeFood 读取图片（支持本地路径和 HTTP URL），调用 VisionModel 分析食物热量，并批量写入 checkins 表。
func analyzeFood(ctx context.Context, in analyzeInput, db *sql.DB, visionModel model.ChatModel) (analyzeResult, error) {
	if in.ThreadID == "" || in.ImagePath == "" {
		return analyzeResult{}, fmt.Errorf("thread_id and image_path required")
	}

	// 1. 读取图片并 base64 编码（支持本地路径和 HTTP URL）
	imgData, mimeType, err := readImage(in.ImagePath)
	if err != nil {
		return analyzeResult{}, fmt.Errorf("read image: %w", err)
	}

	b64 := base64.StdEncoding.EncodeToString(imgData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, b64)

	// 2. 构造多模态消息调用 VisionModel
	msgs := []*schema.Message{
		{
			Role: schema.User,
			MultiContent: []schema.ChatMessagePart{
				{Type: schema.ChatMessagePartTypeText, Text: analyzeFoodPrompt},
				{Type: schema.ChatMessagePartTypeImageURL, ImageURL: &schema.ChatMessageImageURL{URL: dataURL}},
			},
		},
	}

	resp, err := visionModel.Generate(ctx, msgs)
	if err != nil {
		return analyzeResult{}, fmt.Errorf("vision model generate: %w", err)
	}
	recordVisionModelUsage(ctx, db, resp)

	// 3. 解析 JSON 结果（用与 planner.go 同模式的 code-fence 剥离）
	content := extractCodeFenceJSON(resp.Content)

	var result analyzeResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// 解析失败时返回 error（而非静默返回空结果），让 agent 感知并报告用户
		return analyzeResult{}, fmt.Errorf("parse food analysis JSON: %w\nraw: %s", err, content)
	}

	// 4. 批量写入 checkins 表
	for _, food := range result.Foods {
		_, writeErr := recordCheckin(ctx, checkinInput{
			ThreadID: in.ThreadID,
			Category: "diet",
			Content:  food.Name,
			Value:    food.Calories,
		}, db)
		if writeErr != nil {
			continue
		}
	}

	if result.Summary == "" {
		result.Summary = fmt.Sprintf("共识别 %d 种食物，总热量约 %.0f 千卡", len(result.Foods), result.TotalCalories)
	}

	return result, nil
}

func recordVisionModelUsage(ctx context.Context, db *sql.DB, resp *schema.Message) {
	if db == nil || resp == nil || resp.ResponseMeta == nil || resp.ResponseMeta.Usage == nil {
		return
	}
	audit := toolruntime.AuditContextFrom(ctx)
	if audit.RunID == "" || audit.ThreadID == "" {
		return
	}
	usage := resp.ResponseMeta.Usage
	_ = store.AppendModelUsage(ctx, db, store.ModelUsageRecord{
		RunID:            audit.RunID,
		ThreadID:         audit.ThreadID,
		UserID:           audit.UserID,
		Agent:            "vision",
		Model:            "vision",
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		CachedTokens:     usage.PromptTokenDetails.CachedTokens,
		ReasoningTokens:  usage.CompletionTokensDetails.ReasoningTokens,
	})
}

// readImage loads image bytes from a local path or HTTP URL, with a 20MB size limit.
func readImage(path string) ([]byte, string, error) {
	// data: URL — 直接解码 base64
	if strings.HasPrefix(path, "data:") {
		return readImageFromDataURL(path)
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return readImageFromURL(path)
	}
	imgData, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	if len(imgData) > maxImageBytes {
		return nil, "", fmt.Errorf("image too large: %d bytes (max %d)", len(imgData), maxImageBytes)
	}
	return imgData, mimeTypeFromExt(path), nil
}

func readImageFromDataURL(dataURL string) ([]byte, string, error) {
	// data:[<mediatype>][;base64],<data>
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("invalid data URL")
	}
	b64 := parts[1]
	mimeType := "image/jpeg"
	if idx := strings.Index(parts[0], ":"); idx >= 0 {
		mt := parts[0][idx+1:]
		if semi := strings.Index(mt, ";"); semi >= 0 {
			mt = mt[:semi]
		}
		if mt != "" {
			mimeType = mt
		}
	}
	imgData, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, "", fmt.Errorf("decode data URL: %w", err)
	}
	if len(imgData) > maxImageBytes {
		return nil, "", fmt.Errorf("image too large: %d bytes (max %d)", len(imgData), maxImageBytes)
	}
	return imgData, mimeType, nil
}

func readImageFromURL(url string) ([]byte, string, error) {
	if err := security.ValidateExternalURL(url, security.URLPolicyFromConfig()); err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download image %s: HTTP %d", url, resp.StatusCode)
	}

	lr := io.LimitReader(resp.Body, maxImageBytes+1)
	imgData, err := io.ReadAll(lr)
	if err != nil {
		return nil, "", fmt.Errorf("read image body: %w", err)
	}
	if len(imgData) > maxImageBytes {
		return nil, "", fmt.Errorf("image too large: >%d bytes", maxImageBytes)
	}

	// infer MIME type from Content-Type header or URL extension
	ct := resp.Header.Get("Content-Type")
	if ct != "" {
		return imgData, ct, nil
	}
	return imgData, mimeTypeFromExt(url), nil
}

func mimeTypeFromExt(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "image/jpeg"
	}
}

// extractCodeFenceJSON strips markdown ```json ... ``` fences from model output.
// Uses the same pattern as planner.go's extractPlanJSON but handles the edge case
// where the closing ``` is on the same line as content.
func extractCodeFenceJSON(content string) string {
	s := strings.TrimSpace(content)

	if !strings.HasPrefix(s, "```") {
		return s
	}

	// 去掉首行 fence 标记（``` 或 ```json 等）
	if nl := strings.IndexByte(s, '\n'); nl > 0 {
		s = strings.TrimSpace(s[nl+1:])
	} else {
		return s // single line starting with ```, return as-is
	}

	// 去掉结尾的 ```
	if idx := strings.LastIndex(s, "```"); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}

	return s
}
