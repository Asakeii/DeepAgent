package tool

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// analyzeInput 是 analyze_food 工具的入参。
type analyzeInput struct {
	ThreadID  string `json:"thread_id" jsonschema:"required" jsonschema_description:"会话 thread_id"`
	ImagePath string `json:"image_path" jsonschema:"required" jsonschema_description:"本地图片路径（支持 jpg/png/webp）"`
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

// analyzeFood 读取本地图片，调用 VisionModel 分析食物热量，并批量写入 checkins 表。
func analyzeFood(ctx context.Context, in analyzeInput, db *sql.DB, visionModel model.ChatModel) (analyzeResult, error) {
	if in.ThreadID == "" || in.ImagePath == "" {
		return analyzeResult{}, fmt.Errorf("thread_id and image_path required")
	}

	// 1. 读取图片并 base64 编码
	imgData, err := os.ReadFile(in.ImagePath)
	if err != nil {
		return analyzeResult{}, fmt.Errorf("read image %s: %w", in.ImagePath, err)
	}

	ext := strings.ToLower(filepath.Ext(in.ImagePath))
	mimeType := "image/jpeg"
	switch ext {
	case ".png":
		mimeType = "image/png"
	case ".webp":
		mimeType = "image/webp"
	case ".gif":
		mimeType = "image/gif"
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

	// 3. 解析 JSON 结果
	content := resp.Content
	// 模型可能在 JSON 外包了 ```json ... ```，去掉
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		// 去掉第一行 ```json 和最后一行 ```
		if len(lines) > 2 {
			content = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var result analyzeResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// 解析失败，把原始内容作为 summary 返回（降级：至少用户能看到分析结果）
		return analyzeResult{
			Summary: content,
		}, nil
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
			// 写入失败不中断，记录继续
			continue
		}
	}

	if result.Summary == "" {
		result.Summary = fmt.Sprintf("共识别 %d 种食物，总热量约 %.0f 千卡", len(result.Foods), result.TotalCalories)
	}

	return result, nil
}
