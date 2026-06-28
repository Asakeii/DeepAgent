package handler

import (
	"context"
	"crypto/sha1"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"deepAgent/conf"
	"deepAgent/internal/agent"
	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
)

// WeChat 消息 XML 结构
type wechatMsg struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgID        int64    `xml:"MsgId"`
	PicURL       string   `xml:"PicUrl"`
	MediaID      string   `xml:"MediaId"`
}

// WeChat 回复 XML 结构
type wechatReply struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
}

// WechatCallback handles WeChat Official Account server verification (GET) and
// message callbacks (POST). Configure the WeChat backend URL to point at this endpoint.
func WechatCallback(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleWechatVerify(w, r)
	case http.MethodPost:
		handleWechatMessage(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleWechatVerify responds to WeChat's server verification GET request.
// WeChat sends: ?signature=xxx&timestamp=xxx&nonce=xxx&echostr=xxx
func handleWechatVerify(w http.ResponseWriter, r *http.Request) {
	token := os.Getenv("WECHAT_TOKEN")
	if token == "" {
		http.Error(w, "WECHAT_TOKEN not configured", http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()
	signature := q.Get("signature")
	timestamp := q.Get("timestamp")
	nonce := q.Get("nonce")
	echostr := q.Get("echostr")

	if signature == "" || timestamp == "" || nonce == "" || echostr == "" {
		http.Error(w, "missing params", http.StatusBadRequest)
		return
	}

	if checkWechatSignature(token, timestamp, nonce) != signature {
		http.Error(w, "signature mismatch", http.StatusForbidden)
		return
	}

	w.Write([]byte(echostr))
}

// checkWechatSignature computes SHA1(sort(token, timestamp, nonce)) per WeChat spec.
func checkWechatSignature(token, timestamp, nonce string) string {
	parts := []string{token, timestamp, nonce}
	sort.Strings(parts)
	h := sha1.New()
	h.Write([]byte(strings.Join(parts, "")))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// handleWechatMessage processes incoming WeChat message POST requests.
func handleWechatMessage(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	var msg wechatMsg
	if err := xml.Unmarshal(body, &msg); err != nil {
		log.Printf("[wechat] XML parse error: %v, body=%s", err, string(body[:min(len(body), 200)]))
		// XML 解析失败时 msg 为零值，无法构造有效回复（ToUserName/FromUserName 为空）；
		// 返回 HTTP 200 + 空字符串（微信要求必须 200，否则会重试）
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
		return
	}

	threadID := msg.FromUserName
	log.Printf("[wechat] message from %s: %s", threadID, msg.Content)

	// 构建 State，走统一 Coordinator 路由
	userMsg := msg.Content

	// 图片消息特殊处理
	if msg.MsgType == "image" || msg.PicURL != "" {
		userMsg = fmt.Sprintf("打卡早餐 %s", msg.PicURL)
	}

	state := &model.State{
		Messages:                      []*schema.Message{schema.UserMessage(userMsg)},
		Goto:                          consts.Coordinator,
		Locale:                        "zh-CN",
		MaxPlanIterations:             conf.App.Setting.MaxPlanIterations,
		MaxStepNum:                    conf.App.Setting.MaxStepNum,
		AutoAcceptedPlan:              true,
		EnableBackgroundInvestigation: conf.App.Setting.EnableBackgroundInvestigation,
		ThreadID:                      threadID,
	}

	genFunc := func(ctx context.Context) *model.State {
		return state
	}

	runnable, err := agent.Builder(r.Context(), genFunc)
	if err != nil {
		log.Printf("[wechat] build graph: %v", err)
		writeWechatReply(w, msg, "系统繁忙，请稍后再试")
		return
	}

	// collect stream output
	var sb strings.Builder
	outChan := make(chan string)
	done := make(chan struct{})

	go func() {
		for s := range outChan {
			sb.WriteString(s)
		}
		close(done)
	}()

	_, err = runnable.Stream(r.Context(), consts.Coordinator,
		compose.WithCallbacks(&infra.LoggerCallback{
			ID:  threadID,
			Out: outChan,
		}),
	)
	close(outChan)
	<-done

	if err != nil {
		log.Printf("[wechat] graph error: %v", err)
		if sb.Len() > 0 {
			writeWechatReply(w, msg, sb.String())
		} else {
			writeWechatReply(w, msg, "处理出错，请稍后再试")
		}
		return
	}

	replyContent := strings.TrimSpace(sb.String())
	if replyContent == "" {
		replyContent = "已收到"
	}

	writeWechatReply(w, msg, replyContent)
}

func writeWechatReply(w http.ResponseWriter, msg wechatMsg, content string) {
	reply := wechatReply{
		ToUserName:   msg.FromUserName,
		FromUserName: msg.ToUserName,
		CreateTime:   time.Now().Unix(),
		MsgType:      "text",
		Content:      content,
	}

	replyXML, err := xml.Marshal(reply)
	if err != nil {
		http.Error(w, "marshal reply", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write(replyXML)
}
