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
  profiles:
    FAST:
      default_model: "fast-model"
    deep:
      default_model: "deep-model"
      api_key: "dk"
      base_url: "du"
  prices:
    m:
      input_per_million: 1.25
      output_per_million: 5
      cached_input_per_million: 0.25
      reasoning_per_million: 2
setting:
  max_step_num: 3
  run_timeout_seconds: 120
server:
  allowed_origins: ["https://app.example.com"]
  max_body_bytes: 2048
  image_max_bytes: 4096
  image_allowed_types: ["image/png"]
  url_allowed_hosts: ["example.com", "*.trusted.example"]
  url_denied_hosts: ["blocked.example"]
  url_allow_private_networks: true
  api_keys: ["test-key"]
  admin_api_keys: ["admin-key"]
  api_key_principals:
    - key: "worker-key"
      user_id: "service:worker"
      display_name: "Worker"
      admin: false
  oidc:
    issuer_url: "https://issuer.example.com"
    audience: "deepagent-api"
    user_id_claim: "sub"
    display_name_claim: "preferred_username"
    roles_claim: "roles"
    admin_roles: ["deepagent-admin"]
    discovery_timeout_seconds: 8
  rate_limit_per_minute: 60
  sse_heartbeat_seconds: 10
  pdf_renderer_command: "chromium"
  pdf_renderer_args: ["--headless", "--print-to-pdf={{output}}", "{{input}}"]
  pdf_renderer_timeout_seconds: 45
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
	if len(cfg.Model.Profiles) != 2 || cfg.Model.Profiles["FAST"].DefaultModel != "fast-model" || cfg.Model.Profiles["deep"].BaseURL != "du" {
		t.Fatalf("model profiles not parsed: %+v", cfg.Model.Profiles)
	}
	if cfg.Model.Prices["m"].InputPerMillion != 1.25 || cfg.Model.Prices["m"].OutputPerMillion != 5 {
		t.Fatalf("model prices not parsed: %+v", cfg.Model.Prices)
	}
	if cfg.Setting.RunTimeoutSeconds != 120 {
		t.Fatalf("run timeout not parsed: %+v", cfg.Setting)
	}
	if cfg.Server.MaxBodyBytes != 2048 || len(cfg.Server.AllowedOrigins) != 1 || cfg.Server.AllowedOrigins[0] != "https://app.example.com" {
		t.Fatalf("server config not parsed: %+v", cfg.Server)
	}
	if cfg.Server.ImageMaxBytes != 4096 || len(cfg.Server.ImageAllowedTypes) != 1 || cfg.Server.ImageAllowedTypes[0] != "image/png" {
		t.Fatalf("server image config not parsed: %+v", cfg.Server)
	}
	if len(cfg.Server.URLAllowedHosts) != 2 || cfg.Server.URLAllowedHosts[0] != "example.com" || len(cfg.Server.URLDeniedHosts) != 1 || !cfg.Server.URLAllowPrivateNetworks {
		t.Fatalf("server URL policy config not parsed: %+v", cfg.Server)
	}
	if len(cfg.Server.APIKeys) != 1 || cfg.Server.APIKeys[0] != "test-key" || len(cfg.Server.AdminAPIKeys) != 1 || cfg.Server.AdminAPIKeys[0] != "admin-key" || cfg.Server.RateLimitPerMinute != 60 || cfg.Server.SSEHeartbeatSeconds != 10 {
		t.Fatalf("server security config not parsed: %+v", cfg.Server)
	}
	if len(cfg.Server.APIKeyPrincipals) != 1 || cfg.Server.APIKeyPrincipals[0].UserID != "service:worker" || cfg.Server.APIKeyPrincipals[0].DisplayName != "Worker" {
		t.Fatalf("api key principals not parsed: %+v", cfg.Server.APIKeyPrincipals)
	}
	if cfg.Server.OIDC.IssuerURL != "https://issuer.example.com" || cfg.Server.OIDC.Audience != "deepagent-api" || cfg.Server.OIDC.RolesClaim != "roles" || len(cfg.Server.OIDC.AdminRoles) != 1 || cfg.Server.OIDC.DiscoveryTimeoutSeconds != 8 {
		t.Fatalf("oidc config not parsed: %+v", cfg.Server.OIDC)
	}
	if cfg.Server.PDFRendererCommand != "chromium" || len(cfg.Server.PDFRendererArgs) != 3 || cfg.Server.PDFRendererTimeout != 45 {
		t.Fatalf("server pdf renderer config not parsed: %+v", cfg.Server)
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
	if cfg.Server.ImageMaxBytes != DefaultImageMaxBytes || len(cfg.Server.ImageAllowedTypes) == 0 {
		t.Fatalf("image defaults missing: %+v", cfg.Server)
	}
	if cfg.Server.PDFRendererTimeout != 30 {
		t.Fatalf("pdf renderer timeout default missing: %+v", cfg.Server)
	}
	if cfg.Server.OIDC.UserIDClaim != "sub" || cfg.Server.OIDC.DisplayNameClaim != "name" || cfg.Server.OIDC.RolesClaim != "groups" || cfg.Server.OIDC.DiscoveryTimeoutSeconds != 10 {
		t.Fatalf("oidc defaults missing: %+v", cfg.Server.OIDC)
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
