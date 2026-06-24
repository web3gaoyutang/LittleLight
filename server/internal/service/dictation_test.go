package service

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestBuildXFASRAuthURL(t *testing.T) {
	now := time.Date(2026, 6, 19, 5, 0, 0, 0, time.UTC)

	signed, err := BuildXFASRAuthURL("wss://iat-api.xfyun.cn/v2/iat", "test-key", "test-secret", now)
	if err != nil {
		t.Fatalf("build auth url: %v", err)
	}

	parsed, err := url.Parse(signed)
	if err != nil {
		t.Fatalf("parse signed url: %v", err)
	}
	if parsed.Scheme != "wss" || parsed.Host != "iat-api.xfyun.cn" || parsed.Path != "/v2/iat" {
		t.Fatalf("unexpected signed url target: %s", signed)
	}
	query := parsed.Query()
	if query.Get("date") != now.Format("Mon, 02 Jan 2006 15:04:05 GMT") || query.Get("host") != "iat-api.xfyun.cn" {
		t.Fatalf("missing date or host query: %s", signed)
	}
	authRaw, err := base64.StdEncoding.DecodeString(query.Get("authorization"))
	if err != nil {
		t.Fatalf("authorization should be base64: %v", err)
	}
	auth := string(authRaw)
	for _, fragment := range []string{`api_key="test-key"`, `algorithm="hmac-sha256"`, `headers="host date request-line"`, `signature="`} {
		if !strings.Contains(auth, fragment) {
			t.Fatalf("authorization missing %q: %s", fragment, auth)
		}
	}
	if strings.Contains(signed, "test-secret") {
		t.Fatalf("signed URL must not expose api secret: %s", signed)
	}
}

func TestDictationAssemblerAppliesDynamicCorrection(t *testing.T) {
	assembler := NewDictationAssembler()

	text := assembler.ApplyXF(xfIATText{
		SN:  1,
		PGS: "apd",
		WS:  []xfIATWord{{CW: []xfIATCell{{W: "今天小明"}}}},
	})
	if text != "今天小明" {
		t.Fatalf("unexpected first text: %q", text)
	}

	text = assembler.ApplyXF(xfIATText{
		SN:  2,
		PGS: "rpl",
		RG:  []int{1, 1},
		WS:  []xfIATWord{{CW: []xfIATCell{{W: "今天晓明"}}}},
	})
	if text != "今天晓明" {
		t.Fatalf("replacement should update previous segment, got %q", text)
	}

	text = assembler.ApplyXF(xfIATText{
		SN:  3,
		PGS: "apd",
		WS:  []xfIATWord{{CW: []xfIATCell{{W: "上课很专注。"}}}},
	})
	if text != "今天晓明上课很专注。" {
		t.Fatalf("append should preserve corrected text, got %q", text)
	}
}

func TestEventFromXFResultExtractsText(t *testing.T) {
	event := EventFromXFResult(xfIATResponse{
		Code: 0,
		Data: &xfIATResult{
			Status: 1,
			Result: xfIATText{
				SN: 7,
				WS: []xfIATWord{
					{CW: []xfIATCell{{W: "hello"}}},
					{CW: []xfIATCell{{W: " world"}}},
				},
			},
		},
	})

	if event.Type != "partial" || event.SN != 7 || event.Text != "hello world" {
		t.Fatalf("unexpected partial event: %+v", event)
	}
}
