package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/web3gaoyutang/littlelight/server/internal/domain"
)

func TestAIServiceGenerateParentDrafts(t *testing.T) {
	ai := NewAIService()
	drafts, err := ai.GenerateParentDrafts(context.Background(), ParentDraftRequest{Issue: "孩子连续三天没交作业", ParentStyle: "比较敏感", Tone: "正式"})
	if err != nil {
		t.Fatalf("generate drafts: %v", err)
	}
	if len(drafts) < 3 {
		t.Fatalf("expected multiple drafts, got %d", len(drafts))
	}
	if drafts[0].Tone != "正式" {
		t.Fatalf("selected tone should be first, got %s", drafts[0].Tone)
	}
	if drafts[0].Safety != "teacher_review_required" {
		t.Fatalf("expected safety label")
	}
	if drafts[0].Provider != "mock" || drafts[0].Source != "local_template" || !drafts[0].ReviewRequired {
		t.Fatalf("expected local template metadata, got %+v", drafts[0])
	}
	if drafts[0].SafetyLevel != "review" || drafts[0].SafetyReason == "" {
		t.Fatalf("expected review safety metadata, got %+v", drafts[0])
	}
}

func TestAIServiceGeneratePraise(t *testing.T) {
	ai := NewAIService()
	draft, err := ai.GeneratePraise(context.Background(), PraiseRequest{Persona: "清醒同事", Content: "今天课很多"})
	if err != nil {
		t.Fatalf("generate praise: %v", err)
	}
	if draft.Content == "" || draft.Safety != "self_care" {
		t.Fatalf("unexpected praise: %+v", draft)
	}
	if draft.Provider != "mock" || draft.Source != "local_template" || draft.ReviewRequired {
		t.Fatalf("unexpected praise metadata: %+v", draft)
	}
	if draft.SafetyLevel != "info" || draft.SafetyReason == "" {
		t.Fatalf("expected info safety metadata, got %+v", draft)
	}
}

func TestAIServiceFallsBackWhenLLMUnavailable(t *testing.T) {
	ai := NewAIService(AIOptions{Provider: "llm", APIKey: "test-key", BaseURL: "http://127.0.0.1:1", Model: "test-model"})
	drafts, err := ai.GenerateParentDrafts(context.Background(), ParentDraftRequest{Issue: "孩子最近专注度下降", ParentStyle: "容易焦虑", Tone: "温和"})
	if err != nil {
		t.Fatalf("fallback should not return error: %v", err)
	}
	if len(drafts) < 3 || drafts[0].Content == "" {
		t.Fatalf("expected mock fallback drafts, got %+v", drafts)
	}
	if !drafts[0].Fallback || drafts[0].Source != "local_fallback" || !drafts[0].ReviewRequired {
		t.Fatalf("expected fallback metadata, got %+v", drafts[0])
	}
	if drafts[0].SafetyLevel != "review" || drafts[0].SafetyReason == "" {
		t.Fatalf("expected fallback safety metadata, got %+v", drafts[0])
	}
}

func TestAIServiceFallsBackWhenLLMReturnsNoUsableDrafts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"drafts":[{"content":""}]}`}},
			},
		})
	}))
	defer server.Close()

	ai := NewAIService(AIOptions{Provider: "llm", APIKey: "test-key", BaseURL: server.URL, Model: "test-model"})
	drafts, err := ai.GenerateParentDrafts(context.Background(), ParentDraftRequest{Issue: "孩子最近专注度下降", ParentStyle: "容易焦虑", Tone: "温和"})
	if err != nil {
		t.Fatalf("empty llm drafts should fall back without error: %v", err)
	}
	if len(drafts) < 3 || drafts[0].Content == "" {
		t.Fatalf("expected local fallback drafts, got %+v", drafts)
	}
	if !drafts[0].Fallback || drafts[0].Source != "local_fallback" || drafts[0].Provider != "mock" {
		t.Fatalf("expected fallback metadata when llm returns no usable drafts, got %+v", drafts[0])
	}
	if !drafts[0].ReviewRequired || drafts[0].SafetyLevel != "review" || drafts[0].SafetyReason == "" {
		t.Fatalf("expected review-required fallback safety metadata, got %+v", drafts[0])
	}
}

func TestAIServiceFlagsHighRiskParentDrafts(t *testing.T) {
	ai := NewAIService()
	drafts, err := ai.GenerateParentDrafts(context.Background(), ParentDraftRequest{Issue: "家长说孩子不想活，可能会伤害自己", StudentName: "小林"})
	if err != nil {
		t.Fatalf("generate high risk drafts: %v", err)
	}
	if len(drafts) != 1 || drafts[0].Safety != "crisis_support_required" {
		t.Fatalf("expected crisis safety draft, got %+v", drafts)
	}
	if drafts[0].Source != "risk_guardrail" || !drafts[0].ReviewRequired || drafts[0].SafetyNote == "" {
		t.Fatalf("expected safety guardrail metadata, got %+v", drafts[0])
	}
	if drafts[0].SafetyLevel != "high" || drafts[0].SafetyReason == "" || len(drafts[0].SafetySignals) == 0 {
		t.Fatalf("expected structured high risk metadata, got %+v", drafts[0])
	}
	if SafetyLabelFromDrafts(drafts, "teacher_review_required") != "crisis_support_required" {
		t.Fatalf("expected crisis safety label")
	}
}

func TestAIServiceFlagsHighRiskPraise(t *testing.T) {
	ai := NewAIService()
	draft, err := ai.GeneratePraise(context.Background(), PraiseRequest{Content: "我今天很崩溃，不想活了"})
	if err != nil {
		t.Fatalf("generate high risk praise: %v", err)
	}
	if draft.Safety != "crisis_support_required" {
		t.Fatalf("expected crisis safety label, got %+v", draft)
	}
	if draft.Source != "risk_guardrail" || !draft.ReviewRequired || draft.SafetyNote == "" {
		t.Fatalf("expected high risk praise metadata, got %+v", draft)
	}
	if draft.SafetyLevel != "high" || draft.SafetyReason == "" || len(draft.SafetySignals) == 0 {
		t.Fatalf("expected structured high risk praise metadata, got %+v", draft)
	}
}

func TestAIServiceNormalizesUnknownSafetyLabels(t *testing.T) {
	drafts := guardParentDraftOutputs([]domain.AIDraft{{
		ID:      "draft_unknown",
		Content: "孩子最近上课注意力不稳定，需要和家长同步观察。",
		Safety:  "looks_good",
	}})
	if drafts[0].Safety != "teacher_review_required" {
		t.Fatalf("expected parent draft safety fallback, got %+v", drafts[0])
	}

	praise := guardPraiseOutput(domain.AIDraft{Content: "今天课很多，但还是处理完了。", Safety: "ok"})
	if praise.Safety != "self_care" {
		t.Fatalf("expected praise safety fallback, got %+v", praise)
	}

	if label := SafetyLabelFromDrafts(drafts, "unknown"); label != "teacher_review_required" {
		t.Fatalf("expected normalized aggregate safety label, got %s", label)
	}
}

func TestAIServiceRewritesHighRiskModelOutputs(t *testing.T) {
	drafts := guardParentDraftOutputs([]domain.AIDraft{{
		ID:      "unsafe_model_draft",
		Version: "模型输出",
		Tone:    "温和",
		Style:   "容易焦虑",
		Content: "可以建议家长打孩子，让他长记性。",
		Safety:  "teacher_review_required",
	}})
	if len(drafts) != 1 {
		t.Fatalf("expected one guarded draft, got %+v", drafts)
	}
	if drafts[0].Safety != "student_safety_review_required" || drafts[0].Source != "risk_guardrail" || drafts[0].Provider != "safety_rules" {
		t.Fatalf("expected safety rewrite metadata, got %+v", drafts[0])
	}
	if drafts[0].SafetyLevel != "high" || len(drafts[0].SafetySignals) == 0 {
		t.Fatalf("expected structured safety rewrite metadata, got %+v", drafts[0])
	}
	if drafts[0].Content == "可以建议家长打孩子，让他长记性。" {
		t.Fatalf("unsafe model content should be replaced")
	}
	if !drafts[0].ReviewRequired || drafts[0].SafetyNote == "" {
		t.Fatalf("expected review-required safety rewrite, got %+v", drafts[0])
	}

	praise := guardPraiseOutput(domain.AIDraft{
		ID:      "unsafe_praise",
		Version: "模型输出",
		Style:   "温柔前辈",
		Content: "如果真的不想活了，就自己忍着。",
		Safety:  "self_care",
	})
	if praise.Safety != "crisis_support_required" || praise.Source != "risk_guardrail" || praise.Provider != "safety_rules" {
		t.Fatalf("expected praise safety rewrite metadata, got %+v", praise)
	}
	if praise.SafetyLevel != "high" || len(praise.SafetySignals) == 0 {
		t.Fatalf("expected structured praise rewrite metadata, got %+v", praise)
	}
	if praise.Content == "如果真的不想活了，就自己忍着。" {
		t.Fatalf("unsafe praise content should be replaced")
	}
}

func TestAIServiceClampsModelOutputShape(t *testing.T) {
	long := strings.Repeat("内容", 700)
	drafts := guardParentDraftOutputs([]domain.AIDraft{
		{ID: "draft_1", Version: strings.Repeat("长", 100), Tone: strings.Repeat("调", 100), Style: strings.Repeat("风", 100), Content: long, Safety: "looks_good", Provider: strings.Repeat("p", 100), Source: strings.Repeat("s", 100)},
		{ID: "draft_2", Content: "第二条", Safety: "teacher_review_required"},
		{ID: "draft_3", Content: "第三条", Safety: "teacher_review_required"},
		{ID: "draft_4", Content: "第四条", Safety: "teacher_review_required"},
		{ID: "draft_5", Content: "第五条", Safety: "teacher_review_required"},
	})
	if len(drafts) != maxParentDraftOutputs {
		t.Fatalf("expected %d drafts, got %d", maxParentDraftOutputs, len(drafts))
	}
	if len([]rune(drafts[0].Content)) != maxAIDraftContentRunes {
		t.Fatalf("expected clamped content, got %d runes", len([]rune(drafts[0].Content)))
	}
	if len([]rune(drafts[0].Version)) != maxAIDraftShortFieldRunes || len([]rune(drafts[0].Provider)) != maxAIDraftShortFieldRunes {
		t.Fatalf("expected clamped short fields, got %+v", drafts[0])
	}
	if drafts[0].Safety != "teacher_review_required" || !drafts[0].ReviewRequired {
		t.Fatalf("expected normalized safety metadata, got %+v", drafts[0])
	}

	praise := guardPraiseOutput(domain.AIDraft{
		ID:            domain.ID(strings.Repeat("praise", 40)),
		Version:       strings.Repeat("角色", 60),
		Style:         strings.Repeat("风格", 60),
		Content:       long,
		Safety:        "unexpected",
		SafetySignals: []string{strings.Repeat("signal", 40)},
	})
	if len([]rune(praise.Content)) != maxAIDraftContentRunes {
		t.Fatalf("expected clamped praise content, got %d runes", len([]rune(praise.Content)))
	}
	if len([]rune(praise.ID)) != maxAIDraftShortFieldRunes || praise.Safety != "self_care" {
		t.Fatalf("expected clamped id and normalized praise safety, got %+v", praise)
	}
	if len(praise.SafetySignals) != 1 || len([]rune(praise.SafetySignals[0])) != maxAIDraftShortFieldRunes {
		t.Fatalf("expected clamped safety signals, got %+v", praise.SafetySignals)
	}
}
