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
	_ = store.StartReminderTicker(infra.DB)
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

	state := &model.State{
		Messages:                      []*schema.Message{schema.UserMessage(userPrompt)},
		Goto:                          consts.Coordinator,
		Locale:                        "zh-CN",
		MaxPlanIterations:             cfg.Setting.MaxPlanIterations,
		MaxStepNum:                    cfg.Setting.MaxStepNum,
		AutoAcceptedPlan:              true,
		EnableBackgroundInvestigation: cfg.Setting.EnableBackgroundInvestigation,
		ThreadID:                      os.Getenv("DEEPAGENT_THREAD_ID"),
	}

	genFunc := func(ctx context.Context) *model.State {
		return state
	}

	r, err := agent.Builder(context.Background(), genFunc)
	if err != nil {
		log.Fatal(err)
	}

	// 与 deer-go runConsole 对齐：用 Stream 而非 Invoke。
	// 图里的 chat model 节点走流式实现，Coordinator 在 tool call（hand_to_planner）后
	// 不会以字符串形式结束，Invoke 会死等最终输出；Stream 经 callbacks 推送事件才不会卡。
	outChan := make(chan string)
	go func() {
		for out := range outChan {
			fmt.Print(out)
		}
	}()

	_, err = r.Stream(context.Background(), consts.Coordinator,
		compose.WithCallbacks(&infra.LoggerCallback{
			ID:  "console",
			Out: outChan,
		}),
	)
	close(outChan)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println()
}

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
	mux := http.NewServeMux()
	mux.HandleFunc("/chat/stream", handler.ChatStreamEino)

	addr := ":8080"
	log.Printf("deepAgent server listening on %s", addr)
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
