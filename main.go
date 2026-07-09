package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"

	"deepAgent/conf"
	"deepAgent/internal/agent"
	"deepAgent/internal/app"
	"deepAgent/internal/handler"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/observability"
	"deepAgent/internal/scheduler"
	"deepAgent/internal/tool"
)

func main() {
	observability.ConfigureJSONLogger(os.Stderr)
	ctx := context.Background()

	cfg, err := conf.Load(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if err := infra.InitDB(ctx); err != nil {
		log.Fatal(err)
	}
	// Redis 提醒调度器（秒级精度，ZSET 队列 + connRegistry SSE 推送）
	if err := infra.InitRedis(ctx); err != nil {
		log.Printf("[redis] %v — reminders disabled", err)
	}
	if infra.RDB != nil {
		sw := scheduler.Start(ctx, infra.RDB, scheduler.DefaultRegistry)
		defer sw.Stop()
	}
	if err := infra.InitModel(ctx); err != nil {
		log.Fatal(err)
	}
	if err := infra.InitMCP(ctx); err != nil {
		log.Fatal(err)
	}
	if os.Getenv("DEEPAGENT_DEBUG_MCP") == "true" {
		if err := infra.LogMCPTools(ctx); err != nil {
			log.Fatal(err)
		}
	}

	if os.Getenv("DEEPAGENT_MODE") == "server" {
		runServer()
		return
	}

	runCLI(cfg)
}

func runCLI(cfg *conf.Config) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入你的需求： ")
	userPrompt, _ := reader.ReadString('\n')
	userPrompt = strings.TrimSpace(userPrompt)

	msg := schema.UserMessage(userPrompt)
	threadID := os.Getenv("DEEPAGENT_THREAD_ID")

	app.NewChatService().RunStream(context.Background(), model.ChatRequest{
		Messages:                      []*schema.Message{msg},
		ThreadID:                      threadID,
		MaxPlanIterations:             cfg.Setting.MaxPlanIterations,
		MaxStepNum:                    cfg.Setting.MaxStepNum,
		AutoAcceptedPlan:              true,
		EnableBackgroundInvestigation: &cfg.Setting.EnableBackgroundInvestigation,
	}, consoleEventWriter{})
	fmt.Println()
}

type consoleEventWriter struct{}

func (consoleEventWriter) WriteEvent(event string, payload any) error {
	resp, ok := payload.(*model.ChatResp)
	if !ok || resp == nil {
		return nil
	}
	switch event {
	case "message_chunk":
		fmt.Print(resp.Content)
	case "final_message", "message":
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
	case "reminder_scheduled":
		if resp.Content != "" {
			fmt.Printf("\n%s", resp.Content)
		}
	case "reminder":
		if resp.Content != "" {
			fmt.Printf("\n%s", resp.Content)
		}
	case "error":
		if resp.Content != "" {
			fmt.Printf("\n%s", resp.Content)
		}
	case "interrupt":
		fmt.Print("\n检查计划")
	}
	return nil
}

// runCheckin 保留作为独立调试入口（不与 Coordinator 图耦合）。
// 正式路径是 Coordinator 自动路由：用户消息通过 runCLI/runServer 进入。
// 需要手动测试 checkin agent 时设置 DEEPAGENT_MODE=checkin（需恢复 main 中的分支）。
func runCheckin(cfg *conf.Config) {
	ctx := context.Background()

	// thread_id：用环境变量传入，或用一个默认固定值（单用户 console）。
	// 无状态：每次调用从 messages 表加载历史、调用后 append。
	threadID := os.Getenv("DEEPAGENT_THREAD_ID")
	if threadID == "" {
		threadID = "console-default"
	}

	agent, err := agent.NewCheckinAgent(ctx, threadID)
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n你： ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		userInput := strings.TrimSpace(line)
		if userInput == "" {
			continue
		}
		if userInput == "exit" || userInput == "quit" {
			return
		}

		// 加载历史 + 拼当前消息
		history, err := infra.RecentMessagesForCheckin(ctx, threadID, 20)
		if err != nil {
			log.Printf("load history: %v", err)
		}
		msgs := append(history, schema.UserMessage(userInput))

		// 记录用户消息（append 到 messages 表）
		if err := infra.AppendMessageForCheckin(ctx, threadID, string(schema.User), userInput); err != nil {
			log.Printf("append user msg: %v", err)
		}

		// 注入 threadID 到 context（工具通过 ctx 读取）
		agentCtx := tool.WithThreadID(ctx, threadID)
		resp, err := agent.Generate(agentCtx, msgs)
		if err != nil {
			log.Printf("agent error: %v", err)
			continue
		}

		// 记录助手回复
		if err := infra.AppendMessageForCheckin(ctx, threadID, string(schema.Assistant), resp.Content); err != nil {
			log.Printf("append assistant msg: %v", err)
		}

		fmt.Printf("教练：%s\n", resp.Content)
	}
}

func runServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/chat/stream", handler.ChatStreamEino)
	mux.HandleFunc("/wechat/callback", handler.WechatCallback)
	mux.HandleFunc("/v1/chat/completions", handler.OpenAICompatible)
	mux.HandleFunc("/healthz", handler.Healthz)
	mux.HandleFunc("/readyz", handler.Readyz)
	mux.HandleFunc("/share/artifacts", handler.SharedArtifact)
	// 前端静态文件
	mux.HandleFunc("/api/sessions", handler.ListSessions)
	mux.HandleFunc("/api/messages", handler.LoadMessages)
	mux.HandleFunc("/api/run-events", handler.ListRunEvents)
	mux.HandleFunc("/api/runs/cancel", handler.CancelRun)
	mux.HandleFunc("/api/tool-audits", handler.ListToolAudits)
	mux.HandleFunc("/api/metrics/runs", handler.RunMetrics)
	mux.HandleFunc("/api/admin/overview", handler.AdminOverview)
	mux.HandleFunc("/api/memories", handler.Memories)
	mux.HandleFunc("/api/artifacts", handler.ListArtifacts)
	mux.HandleFunc("/api/artifact-exports", handler.ExportArtifact)
	mux.HandleFunc("/api/artifact-shares", handler.ArtifactShares)
	mux.HandleFunc("/api/artifact-citations", handler.ListArtifactCitations)
	mux.HandleFunc("/api/settings", handler.UserSettings)
	mux.HandleFunc("/api/reminders", handler.ListReminders)
	mux.HandleFunc("/api/reminders/cancel", handler.CancelReminder)
	mux.HandleFunc("/api/reminders/toggle", handler.ToggleReminder)
	mux.Handle("/", spaFileServer(frontendDir()))

	addr := ":8741"
	log.Printf("deepAgent server listening on %s", addr)
	if err := http.ListenAndServe(addr, withHTTPGuards(mux)); err != nil {
		log.Fatal(err)
	}
}

func withHTTPGuards(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !applyCORS(w, r) {
			http.Error(w, "origin forbidden", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if requiresAdminProtection(r.URL.Path) && !authorizeAdminRequest(r) {
			http.Error(w, "admin unauthorized", http.StatusUnauthorized)
			return
		}
		if !requiresAdminProtection(r.URL.Path) && requiresAPIProtection(r.URL.Path) && !authorizeAPIRequest(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if requiresAPIProtection(r.URL.Path) && !allowByRateLimit(w, r) {
			return
		}
		if limit := conf.App.Server.MaxBodyBytes; limit > 0 && r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}

func applyCORS(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	for _, allowed := range conf.App.Server.AllowedOrigins {
		if origin == allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-DeepAgent-API-Key, X-DeepAgent-User, X-DeepAgent-Run")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			return true
		}
	}
	return false
}

func requiresAPIProtection(path string) bool {
	return path == "/chat/stream" ||
		path == "/v1/chat/completions" ||
		strings.HasPrefix(path, "/api/")
}

func requiresAdminProtection(path string) bool {
	return strings.HasPrefix(path, "/api/admin/")
}

func authorizeAPIRequest(r *http.Request) bool {
	keys := conf.App.Server.APIKeys
	if len(keys) == 0 {
		return true
	}
	got := requestAPIKey(r)
	if got == "" {
		return false
	}
	for _, key := range keys {
		if subtle.ConstantTimeCompare([]byte(got), []byte(key)) == 1 {
			return true
		}
	}
	return false
}

func authorizeAdminRequest(r *http.Request) bool {
	keys := conf.App.Server.AdminAPIKeys
	if len(keys) == 0 {
		return authorizeAPIRequest(r)
	}
	got := requestAPIKey(r)
	if got == "" {
		return false
	}
	for _, key := range keys {
		if subtle.ConstantTimeCompare([]byte(got), []byte(key)) == 1 {
			return true
		}
	}
	return false
}

func requestAPIKey(r *http.Request) string {
	if raw := strings.TrimSpace(r.Header.Get("Authorization")); raw != "" {
		const prefix = "Bearer "
		if strings.HasPrefix(raw, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(raw, prefix))
		}
	}
	return strings.TrimSpace(r.Header.Get("X-DeepAgent-API-Key"))
}

func allowByRateLimit(w http.ResponseWriter, r *http.Request) bool {
	limit := conf.App.Server.RateLimitPerMinute
	if limit <= 0 || infra.RDB == nil {
		return true
	}
	identity := requestAPIKey(r)
	if identity == "" {
		identity = clientIP(r)
	}
	if identity == "" {
		identity = "unknown"
	}
	window := time.Now().Unix() / 60
	key := fmt.Sprintf("deepagent:rate:%s:%d", hashRateIdentity(identity), window)
	count, err := infra.RDB.Incr(r.Context(), key).Result()
	if err != nil {
		log.Printf("[rate_limit] incr failed: %v", err)
		return true
	}
	if count == 1 {
		_ = infra.RDB.Expire(r.Context(), key, 2*time.Minute).Err()
	}
	if count > int64(limit) {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return false
	}
	return true
}

func hashRateIdentity(identity string) string {
	sum := sha256.Sum256([]byte(identity))
	return hex.EncodeToString(sum[:])
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		if first, _, ok := strings.Cut(forwarded, ","); ok {
			return strings.TrimSpace(first)
		}
		return forwarded
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func frontendDir() string {
	if _, err := os.Stat("frontend/dist/index.html"); err == nil {
		return "frontend/dist"
	}
	return "frontend"
}

func spaFileServer(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			files.ServeHTTP(w, r)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if _, err := os.Stat(dir + "/" + path); err == nil {
			files.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, dir+"/index.html")
	})
}
