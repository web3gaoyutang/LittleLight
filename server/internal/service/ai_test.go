package service

import (
	"context"
	"testing"
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
}
