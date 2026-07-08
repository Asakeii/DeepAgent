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
server:
  allowed_origins: ["https://app.example.com"]
  max_body_bytes: 2048
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
	if cfg.Server.MaxBodyBytes != 2048 || len(cfg.Server.AllowedOrigins) != 1 || cfg.Server.AllowedOrigins[0] != "https://app.example.com" {
		t.Fatalf("server config not parsed: %+v", cfg.Server)
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
}
