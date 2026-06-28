package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"deepAgent/conf"
	"deepAgent/internal/agent"
	"deepAgent/internal/consts"
	"deepAgent/internal/handler"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	srv "deepAgent/internal/server"
	"deepAgent/internal/store"
	"deepAgent/internal/tool"
)

func main() {
	ctx := context.Background()

	cfg, err := conf.Load(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if err := infra.InitDB(ctx); err != nil {
		log.Fatal(err)
	}
	// 启动无状态提醒 ticker（每分钟扫 MySQL 表，抢锁触发到期提醒）
	tickerStop := store.StartReminderTicker(infra.DB)
	defer tickerStop()
	if err := infra.InitModel(ctx); err != nil {
		log.Fatal(err)
	}
	if err := infra.InitMCP(ctx); err != nil {
		log.Fatal(err)
	}
	// 全局编译一次 agent 图（compile-once），后续请求复用
	if err := agent.InitAgent(ctx); err != nil {
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

	// 注入请求级 State
	msg := schema.UserMessage(userPrompt)
	threadID := os.Getenv("DEEPAGENT_THREAD_ID")
	var routedToCheckin bool
	opts := []compose.Option{
		compose.WithStateModifier(func(ctx context.Context, path compose.NodePath, s any) error {
			st := s.(*model.State)
			st.Messages = []*schema.Message{msg}
			st.Locale = "zh-CN"
			st.MaxPlanIterations = cfg.Setting.MaxPlanIterations
			st.MaxStepNum = cfg.Setting.MaxStepNum
			st.ThreadID = threadID
			routedToCheckin = st.RouteToCheckin
			return nil
		}),
	}

	r := agent.GetAgent()

	outChan := make(chan string)
	go func() {
		for out := range outChan {
			fmt.Print(out)
		}
	}()

	var err error
	_, err = r.Stream(context.Background(), consts.Coordinator,
		append(opts, compose.WithCallbacks(&infra.LoggerCallback{
			ID:  "console",
			Out: outChan,
		}))...,
	)
	close(outChan)
	if err != nil {
		log.Fatal(err)
	}
	// Coordinator 标记打卡路由时，切到 checkin agent
	if routedToCheckin {
		resp, cerr := agent.RunCheckin(context.Background(), []*schema.Message{msg}, threadID)
		if cerr != nil {
			log.Printf("[checkin] %v", cerr)
		} else {
			fmt.Println()
			fmt.Println(resp.Content)
		}
		return
	}
	fmt.Println()
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

	agent, err := agent.NewCheckinAgent(ctx)
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
	// 启动 MCP bridge（OpenClaw 通过此端口调用 deepAgent 工具）
	// handle 保留供后续添加 graceful shutdown
	_ = srv.StartMCPServer(":8090")

	mux := http.NewServeMux()
	mux.HandleFunc("/chat/stream", handler.ChatStreamEino)
	mux.HandleFunc("/wechat/callback", handler.WechatCallback)

	addr := ":8080"
	log.Printf("deepAgent server listening on %s (MCP on :8090, wechat on /wechat/callback)", addr)
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
