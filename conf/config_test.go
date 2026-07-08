package conf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadParsesDatabaseAndVision(t *testing.T) {
	dir := t.TempDir()
	ys := []byte(`
database:
  dsn: "user:pass@tcp(127.0.0.1:3306)/db?parseTime=true"
model:
  default_model: "m"
  api_key: "k"
  base_url: "u"
  vision_model:
    default_model: "vm"
    api_key: "vk"
    base_url: "vu"
setting:
  max_step_num: 3
  run_timeout_seconds: 120
server:
  allowed_origins: ["https://app.example.com"]
  max_body_bytes: 2048
  api_keys: ["test-key"]
  rate_limit_per_minute: 60
  sse_heartbeat_seconds: 10
`)
	if err := os.MkdirAll(filepath.Join(dir, "conf"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "conf", "deep-agent.yaml"), ys, 0644); err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	_ = os.Chdir(dir)

	cfg, err := Load(t.Context())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Database.DSN == "" {
		t.Fatal("DSN empty")
	}
	if cfg.Model.VisionModel == nil || cfg.Model.VisionModel.DefaultModel != "vm" {
		t.Fatalf("vision model not parsed: %+v", cfg.Model.VisionModel)
	}
	if cfg.Setting.RunTimeoutSeconds != 120 {
		t.Fatalf("run timeout not parsed: %+v", cfg.Setting)
	}
	if cfg.Server.MaxBodyBytes != 2048 || len(cfg.Server.AllowedOrigins) != 1 || cfg.Server.AllowedOrigins[0] != "https://app.example.com" {
		t.Fatalf("server config not parsed: %+v", cfg.Server)
	}
	if len(cfg.Server.APIKeys) != 1 || cfg.Server.APIKeys[0] != "test-key" || cfg.Server.RateLimitPerMinute != 60 || cfg.Server.SSEHeartbeatSeconds != 10 {
		t.Fatalf("server security config not parsed: %+v", cfg.Server)
	}
}

func TestLoadDefaultsServerConfig(t *testing.T) {
	dir := t.TempDir()
	ys := []byte(`
model:
  default_model: "m"
  api_key: "k"
  base_url: "u"
setting:
  max_step_num: 3
`)
	if err := os.MkdirAll(filepath.Join(dir, "conf"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "conf", "deep-agent.yaml"), ys, 0644); err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	_ = os.Chdir(dir)

	cfg, err := Load(t.Context())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Server.MaxBodyBytes <= 0 || len(cfg.Server.AllowedOrigins) == 0 {
		t.Fatalf("server defaults missing: %+v", cfg.Server)
	}
	if cfg.Server.SSEHeartbeatSeconds != DefaultSSEHeartbeatSeconds {
		t.Fatalf("sse heartbeat default missing: %+v", cfg.Server)
	}
}

func TestServerConfigSSEHeartbeatIntervalDisabled(t *testing.T) {
	cfg := ServerConfig{SSEHeartbeatSeconds: -1}
	if got := cfg.SSEHeartbeatInterval(); got != 0 {
		t.Fatalf("heartbeat interval=%s, want disabled", got)
	}
}

func TestSettingRunTimeout(t *testing.T) {
	if got := (SettingConfig{}).RunTimeout(); got != 0 {
		t.Fatalf("default run timeout=%s, want disabled", got)
	}
	if got := (SettingConfig{RunTimeoutSeconds: 30}).RunTimeout().Seconds(); got != 30 {
		t.Fatalf("run timeout seconds=%v, want 30", got)
	}
}
