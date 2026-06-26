package tool

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// ctxKeyThreadID is the context key for the current session's thread ID.
type ctxKeyThreadID struct{}

// WithThreadID returns a child context with the thread ID set.
// Callers inject this before calling agent.Generate.
func WithThreadID(ctx context.Context, threadID string) context.Context {
	return context.WithValue(ctx, ctxKeyThreadID{}, threadID)
}

// ThreadIDFromCtx returns the thread ID from context, or "" if not set.
func ThreadIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyThreadID{}).(string); ok {
		return v
	}
	return ""
}

// checkinInput 是 record_checkin 工具的入参。
// InferTool 会从 struct tag 自动推导参数 JSON schema。
type checkinInput struct {
	ThreadID string  `json:"thread_id" jsonschema:"required" jsonschema_description:"会话 thread_id"`
	Category string  `json:"category" jsonschema:"required" jsonschema_description:"打卡分类: study/sport/diet/other"`
	Content  string  `json:"content" jsonschema:"required" jsonschema_description:"打卡内容描述"`
	Value    float64 `json:"value" jsonschema_description:"可选数值，如运动公里数、学习小时数"`
}

type checkinOutput struct {
	Message string `json:"message"`
}

// recordCheckin 写一条打卡记录到 checkins 表。
// db 显式注入，便于测试与无状态（不取全局 infra.DB）。
func recordCheckin(ctx context.Context, in checkinInput, db *sql.DB) (checkinOutput, error) {
	if in.ThreadID == "" || in.Category == "" || in.Content == "" {
		return checkinOutput{}, fmt.Errorf("thread_id/category/content required")
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO checkins (thread_id, category, content, value) VALUES (?, ?, ?, ?)`,
		in.ThreadID, in.Category, in.Content, in.Value)
	if err != nil {
		return checkinOutput{}, fmt.Errorf("insert checkin: %w", err)
	}
	return checkinOutput{Message: fmt.Sprintf("已记录 %s 打卡：%s", in.Category, in.Content)}, nil
}

type queryCheckinInput struct {
	ThreadID string `json:"thread_id" jsonschema:"required"`
	Category string `json:"category" jsonschema_description:"可选，按分类过滤；空则全部"`
	Limit    int    `json:"limit" jsonschema_description:"返回条数上限，默认10"`
}

type checkinRecord struct {
	Category  string    `json:"category"`
	Content   string    `json:"content"`
	Value     float64   `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

type queryCheckinOutput struct {
	Records []checkinRecord `json:"records"`
}

func queryCheckin(ctx context.Context, in queryCheckinInput, db *sql.DB) (queryCheckinOutput, error) {
	if in.Limit <= 0 {
		in.Limit = 10
	}
	q := `SELECT category, content, COALESCE(value,0), created_at FROM checkins WHERE thread_id = ?`
	args := []any{in.ThreadID}
	if in.Category != "" {
		q += ` AND category = ?`
		args = append(args, in.Category)
	}
	q += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, in.Limit)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return queryCheckinOutput{}, fmt.Errorf("query checkins: %w", err)
	}
	defer rows.Close()

	var out queryCheckinOutput
	for rows.Next() {
		var r checkinRecord
		if err := rows.Scan(&r.Category, &r.Content, &r.Value, &r.CreatedAt); err != nil {
			return queryCheckinOutput{}, err
		}
		out.Records = append(out.Records, r)
	}
	return out, rows.Err()
}

type summaryInput struct {
	ThreadID string `json:"thread_id" jsonschema:"required"`
	Days     int    `json:"days" jsonschema_description:"最近 N 天，默认7"`
}

type summaryOutput struct {
	Summary string `json:"summary"`
}

// getSummary 汇总某 thread 最近 N 天的打卡情况，文本交给 agent 润色。
func getSummary(ctx context.Context, in summaryInput, db *sql.DB) (summaryOutput, error) {
	if in.Days <= 0 {
		in.Days = 7
	}
	q := `SELECT category, COUNT(*) FROM checkins WHERE thread_id = ? AND created_at >= DATE_SUB(NOW(), INTERVAL ? DAY) GROUP BY category`
	rows, err := db.QueryContext(ctx, q, in.ThreadID, in.Days)
	if err != nil {
		return summaryOutput{}, fmt.Errorf("summary: %w", err)
	}
	defer rows.Close()

	var parts []string
	for rows.Next() {
		var cat string
		var cnt int
		if err := rows.Scan(&cat, &cnt); err != nil {
			return summaryOutput{}, err
		}
		parts = append(parts, fmt.Sprintf("%s:%d", cat, cnt))
	}
	if len(parts) == 0 {
		return summaryOutput{Summary: "最近没有打卡记录"}, nil
	}
	return summaryOutput{Summary: fmt.Sprintf("最近%d天打卡统计: %s", in.Days, joinStrings(parts, ", "))}, nil
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	r := ss[0]
	for _, s := range ss[1:] {
		r += sep + s
	}
	return r
}

// CheckinTools 返回 checkin agent 使用的全部工具。
// db 和 visionModel 由调用方注入；threadID 通过 context.WithValue(ctxKeyThreadID{}, tid) 在调用时传入。
func CheckinTools(ctx context.Context, db *sql.DB, visionModel model.ChatModel) ([]tool.BaseTool, error) {
	var tools []tool.BaseTool

	rc, err := utils.InferTool("record_checkin",
		"记录一条用户打卡（学习/运动/饮食等）",
		func(ctx context.Context, in checkinInput) (checkinOutput, error) {
			if tid := ThreadIDFromCtx(ctx); tid != "" {
				in.ThreadID = tid
			}
			return recordCheckin(ctx, in, db)
		})
	if err != nil {
		return nil, fmt.Errorf("infer record_checkin: %w", err)
	}
	tools = append(tools, rc)

	qc, err := utils.InferTool("query_checkin",
		"查询用户历史打卡记录",
		func(ctx context.Context, in queryCheckinInput) (queryCheckinOutput, error) {
			if tid := ThreadIDFromCtx(ctx); tid != "" {
				in.ThreadID = tid
			}
			return queryCheckin(ctx, in, db)
		})
	if err != nil {
		return nil, fmt.Errorf("infer query_checkin: %w", err)
	}
	tools = append(tools, qc)

	gs, err := utils.InferTool("get_summary",
		"汇总用户最近若干天的打卡情况",
		func(ctx context.Context, in summaryInput) (summaryOutput, error) {
			if tid := ThreadIDFromCtx(ctx); tid != "" {
				in.ThreadID = tid
			}
			return getSummary(ctx, in, db)
		})
	if err != nil {
		return nil, fmt.Errorf("infer get_summary: %w", err)
	}
	tools = append(tools, gs)

	af, err := utils.InferTool("analyze_food",
		"分析食物图片，识别所有食物的名称、份量和热量，并自动记录到打卡。用户发食物照片时调用此工具。",
		func(ctx context.Context, in analyzeInput) (analyzeResult, error) {
			if tid := ThreadIDFromCtx(ctx); tid != "" {
				in.ThreadID = tid
			}
			return analyzeFood(ctx, in, db, visionModel)
		})
	if err != nil {
		return nil, fmt.Errorf("infer analyze_food: %w", err)
	}
	tools = append(tools, af)

	cr, err := utils.InferTool("create_reminder",
		"创建一个定时提醒（每天/每周提醒等），用户说'每天8点提醒我喝水'时调用",
		makeCreateReminder(db))
	if err != nil {
		return nil, fmt.Errorf("infer create_reminder: %w", err)
	}
	tools = append(tools, cr)

	lr, err := utils.InferTool("list_reminders",
		"列出用户当前的所有提醒",
		makeListReminders(db))
	if err != nil {
		return nil, fmt.Errorf("infer list_reminders: %w", err)
	}
	tools = append(tools, lr)

	dr, err := utils.InferTool("delete_reminder",
		"删除一条提醒（需要提醒的 id，先 list_reminders 获取）",
		makeDeleteReminder(db))
	if err != nil {
		return nil, fmt.Errorf("infer delete_reminder: %w", err)
	}
	tools = append(tools, dr)

	return tools, nil
}
