package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/web3gaoyutang/littlelight/server/internal/config"
)

func TestValidateProductionAuthConfigAllowsNonProductionDevelopment(t *testing.T) {
	for _, appEnv := range []string{"local", "docker", "staging"} {
		t.Run(appEnv, func(t *testing.T) {
			err := validateProductionAuthConfig(config.Config{
				AppEnv:        appEnv,
				AllowDevUser:  true,
				AllowMockAuth: true,
			})
			if err != nil {
				t.Fatalf("%s development config should be allowed, got %v", appEnv, err)
			}
		})
	}
}

func TestValidateProductionAuthConfigRequiresRealAuthOutsideLocal(t *testing.T) {
	valid := productionAuthConfig()
	tests := []struct {
		name    string
		mutate  func(*config.Config)
		message string
	}{
		{
			name: "dev user header disabled",
			mutate: func(cfg *config.Config) {
				cfg.AllowDevUser = true
			},
			message: "AUTH_ALLOW_DEV_USER",
		},
		{
			name: "mock login disabled",
			mutate: func(cfg *config.Config) {
				cfg.AllowMockAuth = true
			},
			message: "AUTH_ALLOW_MOCK_LOGIN",
		},
		{
			name: "cors origins required",
			mutate: func(cfg *config.Config) {
				cfg.CORSOrigins = nil
			},
			message: "CORS_ALLOWED_ORIGINS",
		},
		{
			name: "wildcard cors rejected",
			mutate: func(cfg *config.Config) {
				cfg.CORSOrigins = []string{"*"}
			},
			message: "CORS_ALLOWED_ORIGINS",
		},
		{
			name: "strong session secret required",
			mutate: func(cfg *config.Config) {
				cfg.SessionSecret = "short"
			},
			message: "SESSION_SECRET",
		},
		{
			name: "placeholder session secret rejected",
			mutate: func(cfg *config.Config) {
				cfg.SessionSecret = "change-me-in-deploy"
			},
			message: "SESSION_SECRET",
		},
		{
			name: "wechat app id required",
			mutate: func(cfg *config.Config) {
				cfg.WechatAppID = ""
			},
			message: "WECHAT_APP_ID",
		},
		{
			name: "wechat secret required",
			mutate: func(cfg *config.Config) {
				cfg.WechatSecret = ""
			},
			message: "WECHAT_APP_SECRET",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := valid
			test.mutate(&cfg)
			err := validateProductionAuthConfig(cfg)
			if err == nil {
				t.Fatalf("expected validation error")
			}
			if !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected error mentioning %q, got %v", test.message, err)
			}
		})
	}
}

func TestValidateProductionAuthConfigAcceptsProductionReadyConfig(t *testing.T) {
	if err := validateProductionAuthConfig(productionAuthConfig()); err != nil {
		t.Fatalf("production-ready auth config should pass, got %v", err)
	}

	cfg := productionAuthConfig()
	cfg.AppEnv = "prod"
	if err := validateProductionAuthConfig(cfg); err != nil {
		t.Fatalf("prod alias should use the same strict auth checks, got %v", err)
	}
}

func TestNewHTTPServerConfiguresTimeouts(t *testing.T) {
	handler := http.NewServeMux()
	server := newHTTPServer(":0", handler)

	if server.Addr != ":0" {
		t.Fatalf("unexpected server addr: %s", server.Addr)
	}
	if server.Handler != handler {
		t.Fatalf("server should use the provided handler")
	}
	if server.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("unexpected ReadHeaderTimeout: %s", server.ReadHeaderTimeout)
	}
	if server.ReadTimeout != 15*time.Second {
		t.Fatalf("unexpected ReadTimeout: %s", server.ReadTimeout)
	}
	if server.WriteTimeout != 30*time.Second {
		t.Fatalf("unexpected WriteTimeout: %s", server.WriteTimeout)
	}
	if server.IdleTimeout != 60*time.Second {
		t.Fatalf("unexpected IdleTimeout: %s", server.IdleTimeout)
	}
}

func productionAuthConfig() config.Config {
	return config.Config{
		AppEnv:        "production",
		SessionSecret: strings.Repeat("s", 32),
		WechatAppID:   "wx-test-app",
		WechatSecret:  "wx-test-secret",
		AllowDevUser:  false,
		AllowMockAuth: false,
		CORSOrigins:   []string{"https://app.example.com"},
	}
}
