package scheduler

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Tools returns the reminder tools the agent can call.
// rdb is the Redis client; threadID is injected by the server so the model
// cannot accidentally schedule into another session.
func Tools(ctx context.Context, rdb *redis.Client, threadID string) ([]tool.BaseTool, error) {
	s, err := utils.InferTool("schedule_reminder",
		"创建一个定时提醒/待办提醒（支持绝对时间、延时或cron）。当用户说提醒我、记得、记一下、别忘了、叫我、喊我，并带未来时间时调用本工具，不要记录为打卡。优先填写 fire_at_text（例如'今晚九点'、'明天上午10点'）；明确秒数延时才填写delay_seconds；重复提醒填写cron_expression，三者选一。\n示例：fire_at_text='今晚九点'；delay_seconds=30表示30秒后提醒；cron_expression='0 8 * * *'表示每天8点。",
		func(ctx context.Context, in scheduleInput) (scheduleOutput, error) {
			in.ThreadID = threadID
			return scheduleReminder(ctx, rdb, in)
		})
	if err != nil {
		return nil, fmt.Errorf("infer schedule_reminder: %w", err)
	}

	c, err := utils.InferTool("cancel_reminder",
		"取消一个未触发的提醒。传入 reminder_id。",
		func(ctx context.Context, in cancelInput) (cancelOutput, error) {
			in.ThreadID = threadID
			return cancelReminder(ctx, rdb, in)
		})
	if err != nil {
		return nil, fmt.Errorf("infer cancel_reminder: %w", err)
	}

	l, err := utils.InferTool("list_reminders",
		"列出当前会话的所有待触发提醒。",
		func(ctx context.Context, in listInput) (listOutput, error) {
			in.ThreadID = threadID
			return listReminders(ctx, rdb, in)
		})
	if err != nil {
		return nil, fmt.Errorf("infer list_reminders: %w", err)
	}

	return []tool.BaseTool{s, c, l}, nil
}

// ---- types ----

type scheduleInput struct {
	Message        string `json:"message" jsonschema:"required" jsonschema_description:"提醒内容"`
	FireAtText     string `json:"fire_at_text" jsonschema_description:"自然语言触发时间，如'今晚九点'、'明天上午10点'、'后天下午3:30'。优先使用此字段"`
	DelaySeconds   int    `json:"delay_seconds" jsonschema_description:"延时秒数，如30表示30秒后提醒。与cron_expression二选一"`
	CronExpression string `json:"cron_expression" jsonschema_description:"cron表达式，如'0 8 * * *'表示每天8点。与delay_seconds二选一"`
	ThreadID       string `json:"thread_id" jsonschema_description:"会话ID，由服务端注入，通常无需填写"`
}

type scheduleOutput struct {
	ReminderID string `json:"reminder_id"`
	FireAt     string `json:"fire_at"`
	FireAtUnix int64  `json:"fire_at_unix"`
	Cron       string `json:"cron,omitempty"`
	Recurring  bool   `json:"recurring"`
	Message    string `json:"message"`
}

type cancelInput struct {
	ReminderID string `json:"reminder_id" jsonschema:"required" jsonschema_description:"要取消的提醒ID"`
	ThreadID   string `json:"thread_id" jsonschema_description:"会话ID，由服务端注入，通常无需填写"`
}

type cancelOutput struct {
	Message string `json:"message"`
}

type listInput struct {
	ThreadID string `json:"thread_id" jsonschema_description:"会话ID，由服务端注入，通常无需填写"`
}

type reminderItem struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	FireAt  string `json:"fire_at"`
	Cron    string `json:"cron"`
}

type listOutput struct {
	Reminders []reminderItem `json:"reminders"`
}

// ---- logic ----

func scheduleReminder(ctx context.Context, rdb *redis.Client, in scheduleInput) (scheduleOutput, error) {
	if in.Message == "" {
		return scheduleOutput{}, fmt.Errorf("message is required")
	}
	if in.ThreadID == "" {
		return scheduleOutput{}, fmt.Errorf("thread_id is required")
	}

	var fireAt int64
	var recurring bool

	switch {
	case in.FireAtText != "":
		next, err := parseFireAtText(in.FireAtText, time.Now())
		if err != nil {
			return scheduleOutput{}, err
		}
		fireAt = next.Unix()
	case in.DelaySeconds > 0:
		fireAt = time.Now().Add(time.Duration(in.DelaySeconds) * time.Second).Unix()
	case in.CronExpression != "":
		sched, err := CronParser.Parse(in.CronExpression)
		if err != nil {
			return scheduleOutput{}, fmt.Errorf("invalid cron %q: %w", in.CronExpression, err)
		}
		next := sched.Next(reminderNow())
		if next.IsZero() {
			return scheduleOutput{}, fmt.Errorf("cron %q has no future fire time", in.CronExpression)
		}
		fireAt = next.Unix()
		recurring = true
	default:
		return scheduleOutput{}, fmt.Errorf("fire_at_text, delay_seconds, or cron_expression required")
	}

	r := Reminder{
		ID:        uuid.NewString(),
		ThreadID:  in.ThreadID,
		Message:   in.Message,
		FireAt:    fireAt,
		Cron:      in.CronExpression,
		Recurring: recurring,
	}

	if err := Schedule(ctx, rdb, r); err != nil {
		return scheduleOutput{}, fmt.Errorf("schedule: %w", err)
	}
	EmitEvent(ctx, eventFromReminder(r, "scheduled"))

	typ := "延时提醒"
	if recurring {
		typ = "重复提醒"
	}
	return scheduleOutput{
		ReminderID: r.ID,
		FireAt:     time.Unix(fireAt, 0).In(reminderLocation()).Format("15:04:05"),
		FireAtUnix: fireAt,
		Cron:       r.Cron,
		Recurring:  r.Recurring,
		Message:    fmt.Sprintf("已设置%s：%s", typ, in.Message),
	}, nil
}

func parseFireAtText(text string, now time.Time) (time.Time, error) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err == nil {
		now = now.In(loc)
	}
	dayOffset := 0
	switch {
	case strings.Contains(text, "后天"):
		dayOffset = 2
	case strings.Contains(text, "明天"):
		dayOffset = 1
	}
	hour, minute, ok := parseHourMinuteText(text)
	if !ok {
		return time.Time{}, fmt.Errorf("cannot parse fire_at_text %q", text)
	}
	if (strings.Contains(text, "下午") || strings.Contains(text, "晚上") || strings.Contains(text, "今晚")) && hour < 12 {
		hour += 12
	}
	if strings.Contains(text, "中午") && hour < 11 {
		hour += 12
	}
	next := time.Date(now.Year(), now.Month(), now.Day()+dayOffset, hour, minute, 0, 0, now.Location())
	if dayOffset == 0 && !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next, nil
}

func parseHourMinuteText(text string) (int, int, bool) {
	re := regexp.MustCompile(`([0-2]?\d)[:：点时]([0-5]\d)?`)
	if matches := re.FindStringSubmatch(text); len(matches) > 0 {
		hour, ok := atoiSmall(matches[1])
		if !ok || hour > 23 {
			return 0, 0, false
		}
		minute := 0
		if matches[2] != "" {
			parsed, ok := atoiSmall(matches[2])
			if !ok || parsed > 59 {
				return 0, 0, false
			}
			minute = parsed
		}
		return hour, minute, true
	}
	reChinese := regexp.MustCompile(`([一二两三四五六七八九十]{1,3})[点时]`)
	if matches := reChinese.FindStringSubmatch(text); len(matches) > 0 {
		hour, ok := chineseHour(matches[1])
		return hour, 0, ok
	}
	return 0, 0, false
}

func atoiSmall(s string) (int, bool) {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, false
		}
		n = n*10 + int(ch-'0')
	}
	return n, true
}

func chineseHour(s string) (int, bool) {
	if s == "两" {
		return 2, true
	}
	digit := map[rune]int{'一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9}
	runes := []rune(s)
	if len(runes) == 1 {
		if runes[0] == '十' {
			return 10, true
		}
		v, ok := digit[runes[0]]
		return v, ok
	}
	if runes[0] == '十' {
		v, ok := digit[runes[1]]
		return 10 + v, ok
	}
	if len(runes) == 2 && runes[1] == '十' {
		v, ok := digit[runes[0]]
		return v * 10, ok
	}
	if len(runes) == 3 && runes[1] == '十' {
		tens, ok1 := digit[runes[0]]
		ones, ok2 := digit[runes[2]]
		return tens*10 + ones, ok1 && ok2
	}
	return 0, false
}

func cancelReminder(ctx context.Context, rdb *redis.Client, in cancelInput) (cancelOutput, error) {
	if in.ReminderID == "" || in.ThreadID == "" {
		return cancelOutput{}, fmt.Errorf("reminder_id and thread_id required")
	}
	if err := Cancel(ctx, rdb, in.ThreadID, in.ReminderID); err != nil {
		return cancelOutput{}, fmt.Errorf("cancel: %w", err)
	}
	return cancelOutput{Message: "已取消提醒 " + in.ReminderID[:8]}, nil
}

func listReminders(ctx context.Context, rdb *redis.Client, in listInput) (listOutput, error) {
	if in.ThreadID == "" {
		return listOutput{}, fmt.Errorf("thread_id required")
	}
	reminders, err := List(ctx, rdb, in.ThreadID)
	if err != nil {
		return listOutput{}, fmt.Errorf("list: %w", err)
	}
	var items []reminderItem
	for _, r := range reminders {
		items = append(items, reminderItem{
			ID:      r.ID[:8],
			Message: r.Message,
			FireAt:  time.Unix(r.FireAt, 0).In(reminderLocation()).Format("2006-01-02 15:04:05"),
			Cron:    r.Cron,
		})
	}
	if items == nil {
		items = []reminderItem{}
	}
	return listOutput{Reminders: items}, nil
}
