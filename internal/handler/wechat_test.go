package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWechatPostRejectsInvalidSignature(t *testing.T) {
	t.Setenv("WECHAT_TOKEN", "token")
	req := httptest.NewRequest(http.MethodPost, "/wechat/callback?signature=bad&timestamp=1&nonce=2", strings.NewReader("<xml></xml>"))
	rec := httptest.NewRecorder()

	WechatCallback(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestWechatPostAcceptsValidSignatureBeforeReadingXML(t *testing.T) {
	t.Setenv("WECHAT_TOKEN", "token")
	signature := checkWechatSignature("token", "1", "2")
	req := httptest.NewRequest(http.MethodPost, "/wechat/callback?signature="+signature+"&timestamp=1&nonce=2", strings.NewReader("not-xml"))
	rec := httptest.NewRecorder()

	WechatCallback(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "success" {
		t.Fatalf("body=%q, want success", got)
	}
}

func TestWechatGetRejectsMissingToken(t *testing.T) {
	t.Setenv("WECHAT_TOKEN", "")
	req := httptest.NewRequest(http.MethodGet, "/wechat/callback?signature=s&timestamp=1&nonce=2&echostr=ok", nil)
	rec := httptest.NewRecorder()

	WechatCallback(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
