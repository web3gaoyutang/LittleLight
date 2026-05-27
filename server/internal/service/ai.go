package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/web3gaoyutang/littlelight/server/internal/domain"
)

type AIService struct{}

type ParentDraftRequest struct {
	Issue       string `json:"issue"`
	ParentStyle string `json:"parentStyle"`
	Tone        string `json:"tone"`
	StudentName string `json:"studentName"`
}

type PraiseRequest struct {
	Persona string `json:"persona"`
	Content string `json:"content"`
	Mood    string `json:"mood"`
}

func NewAIService() *AIService {
	return &AIService{}
}

func (s *AIService) GenerateParentDrafts(ctx context.Context, req ParentDraftRequest) ([]domain.AIDraft, error) {
	issue := strings.TrimSpace(req.Issue)
	if issue == "" {
		issue = "家长询问孩子最近在课堂上的专注度情况，希望老师给出温和、可执行的建议。"
	}
	style := defaultString(req.ParentStyle, "容易焦虑")
	tone := defaultString(req.Tone, "温和")
	versions := []string{tone, "正式", "简短", "坚定但礼貌"}
	seen := map[string]bool{}
	drafts := make([]domain.AIDraft, 0, len(versions))
	for _, version := range versions {
		if seen[version] {
			continue
		}
		seen[version] = true
		drafts = append(drafts, domain.AIDraft{ID: domain.ID("draft_" + normalize(version)), Version: version, Tone: version, Style: style, Safety: "teacher_review_required", Content: parentDraft(issue, style, version)})
	}
	return drafts, nil
}

func (s *AIService) GeneratePraise(ctx context.Context, req PraiseRequest) (domain.AIDraft, error) {
	persona := defaultString(req.Persona, "温柔前辈")
	content := strings.TrimSpace(req.Content)
	if content == "" {
		content = "今天处理了课程、待办和家长沟通"
	}
	prefix := "先认真夸你一下："
	if strings.Contains(persona, "主任") {
		prefix = "我看见的是一个很负责的老师："
	}
	if strings.Contains(persona, "同事") || strings.Contains(persona, "闺蜜") {
		prefix = "说句实在的，"
	}
	return domain.AIDraft{ID: "praise_1", Version: persona, Tone: "supportive", Style: persona, Safety: "self_care", Content: fmt.Sprintf("%s你今天处理了“%s”，这不是小事。先把标准降到可持续，剩下的事情可以一件一件来。", prefix, content)}, nil
}

func parentDraft(issue, style, tone string) string {
	opening := map[string]string{
		"容易焦虑":   "先回应您的担心，也谢谢您这么细致地关注孩子。",
		"比较敏感":   "我会尽量具体地说明，避免让您只看到负面的部分。",
		"沟通积极":   "谢谢您主动同步，这会让我们更快形成一致的支持方式。",
		"关注成绩":   "我先把表现和后续提升路径分开说明，方便您判断重点。",
	}
	body := map[string]string{
		"温和":    "建议这两天先观察作息、作业启动速度和课堂后的疲惫感，不急着批评，多给一个可完成的小目标。",
		"正式":    "建议本周以稳定作息、明确任务边界和固定复盘时间为主，我会在周五继续同步课堂状态。",
		"简短":    "这周先抓一个重点：稳定节奏和完成订正。我会继续观察并及时反馈。",
		"坚定但礼貌": "目前最需要先把基础节奏稳住，家庭端建议减少临时加码，按约定计划执行后再看变化。",
	}
	return opening[defaultStyle(style)] + "关于“" + issue + "”，" + body[defaultTone(tone)] + "如果您方便，今晚可以先记录孩子的状态，我明天结合课堂表现再反馈一次。"
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func defaultStyle(value string) string {
	if _, ok := map[string]bool{"容易焦虑": true, "比较敏感": true, "沟通积极": true, "关注成绩": true}[value]; ok {
		return value
	}
	return "容易焦虑"
}

func defaultTone(value string) string {
	if _, ok := map[string]bool{"温和": true, "正式": true, "简短": true, "坚定但礼貌": true}[value]; ok {
		return value
	}
	return "温和"
}

func normalize(value string) string {
	return strings.NewReplacer(" ", "_", "但", "_", "礼貌", "polite", "温和", "warm", "正式", "formal", "简短", "short", "坚定", "firm").Replace(value)
}
