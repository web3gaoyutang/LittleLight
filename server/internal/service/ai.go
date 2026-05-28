package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/web3gaoyutang/littlelight/server/internal/domain"
)

type AIService struct {
	provider string
	apiKey   string
	baseURL  string
	model    string
	client   *http.Client
}

type AIOptions struct {
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
}

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

func NewAIService(options ...AIOptions) *AIService {
	opts := AIOptions{Provider: "mock", Model: "gpt-4o-mini"}
	if len(options) > 0 {
		opts = options[0]
	}
	opts.Provider = strings.TrimSpace(opts.Provider)
	if opts.Provider == "" {
		opts.Provider = "mock"
		if strings.TrimSpace(opts.APIKey) != "" && strings.TrimSpace(opts.BaseURL) != "" {
			opts.Provider = "llm"
		}
	}
	if opts.Model == "" {
		opts.Model = "gpt-4o-mini"
	}
	return &AIService{
		provider: opts.Provider,
		apiKey:   strings.TrimSpace(opts.APIKey),
		baseURL:  strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/"),
		model:    strings.TrimSpace(opts.Model),
		client:   &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *AIService) GenerateParentDrafts(ctx context.Context, req ParentDraftRequest) ([]domain.AIDraft, error) {
	if s.useLLM() {
		if drafts, err := s.generateParentDraftsWithLLM(ctx, req); err == nil && len(drafts) > 0 {
			return drafts, nil
		} else if err != nil {
			log.Printf("llm parent drafts failed, using mock fallback: %v", err)
		}
	}
	return s.generateParentDraftsMock(req), nil
}

func (s *AIService) generateParentDraftsMock(req ParentDraftRequest) []domain.AIDraft {
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
	return drafts
}

func (s *AIService) GeneratePraise(ctx context.Context, req PraiseRequest) (domain.AIDraft, error) {
	if s.useLLM() {
		if draft, err := s.generatePraiseWithLLM(ctx, req); err == nil && draft.Content != "" {
			return draft, nil
		} else if err != nil {
			log.Printf("llm praise failed, using mock fallback: %v", err)
		}
	}
	return s.generatePraiseMock(req), nil
}

func (s *AIService) generatePraiseMock(req PraiseRequest) domain.AIDraft {
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
	return domain.AIDraft{ID: "praise_1", Version: persona, Tone: "supportive", Style: persona, Safety: "self_care", Content: fmt.Sprintf("%s你今天处理了“%s”，这不是小事。先把标准降到可持续，剩下的事情可以一件一件来。", prefix, content)}
}

func (s *AIService) useLLM() bool {
	provider := strings.ToLower(s.provider)
	return (provider == "llm" || provider == "openai" || provider == "openai-compatible" || provider == "qwen") && s.apiKey != "" && s.baseURL != ""
}

func (s *AIService) generateParentDraftsWithLLM(ctx context.Context, req ParentDraftRequest) ([]domain.AIDraft, error) {
	style := defaultString(req.ParentStyle, "容易焦虑")
	tone := defaultString(req.Tone, "温和")
	issue := defaultString(req.Issue, "家长询问孩子最近在课堂上的专注度情况，希望老师给出温和、可执行的建议。")
	prompt := fmt.Sprintf(`请为老师生成 4 条可选家校沟通草稿。
要求：
1. 家长风格：%s。
2. 首选语气：%s，另外补充正式、简短、坚定但礼貌的版本；如果重复则换成更合适的版本。
3. 问题背景：%s。
4. 必须温和、具体、可执行，避免医学诊断和承诺性判断。
5. 只返回 JSON，格式为 {"drafts":[{"version":"","tone":"","style":"","content":"","safety":"teacher_review_required"}]}`, style, tone, issue)
	var payload struct {
		Drafts []struct {
			Version string `json:"version"`
			Tone    string `json:"tone"`
			Style   string `json:"style"`
			Content string `json:"content"`
			Safety  string `json:"safety"`
		} `json:"drafts"`
	}
	if err := s.chatJSON(ctx, "你是面向教师的家校沟通助手，只输出可审核的 JSON。", prompt, &payload); err != nil {
		return nil, err
	}
	drafts := make([]domain.AIDraft, 0, len(payload.Drafts))
	for i, item := range payload.Drafts {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		version := defaultString(item.Version, fmt.Sprintf("版本%d", i+1))
		drafts = append(drafts, domain.AIDraft{
			ID:      domain.ID(fmt.Sprintf("llm_draft_%d", i+1)),
			Version: version,
			Tone:    defaultString(item.Tone, version),
			Style:   defaultString(item.Style, style),
			Content: content,
			Safety:  defaultString(item.Safety, "teacher_review_required"),
		})
	}
	return drafts, nil
}

func (s *AIService) generatePraiseWithLLM(ctx context.Context, req PraiseRequest) (domain.AIDraft, error) {
	persona := defaultString(req.Persona, "温柔前辈")
	content := defaultString(req.Content, "今天处理了课程、待办和家长沟通")
	mood := defaultString(req.Mood, "warm")
	prompt := fmt.Sprintf(`请生成一条给老师的 AI 夸夸。
要求：
1. 角色口吻：%s。
2. 老师输入：%s。
3. 当前情绪：%s。
4. 内容要具体、有边界感、不过度鸡血，避免医疗建议。
5. 只返回 JSON，格式为 {"version":"","tone":"supportive","style":"","content":"","safety":"self_care"}`, persona, content, mood)
	var payload struct {
		Version string `json:"version"`
		Tone    string `json:"tone"`
		Style   string `json:"style"`
		Content string `json:"content"`
		Safety  string `json:"safety"`
	}
	if err := s.chatJSON(ctx, "你是面向教师的情绪支持助手，只输出可审核的 JSON。", prompt, &payload); err != nil {
		return domain.AIDraft{}, err
	}
	if strings.TrimSpace(payload.Content) == "" {
		return domain.AIDraft{}, fmt.Errorf("empty llm praise content")
	}
	return domain.AIDraft{
		ID:      "llm_praise_1",
		Version: defaultString(payload.Version, persona),
		Tone:    defaultString(payload.Tone, "supportive"),
		Style:   defaultString(payload.Style, persona),
		Content: payload.Content,
		Safety:  defaultString(payload.Safety, "self_care"),
	}, nil
}

func (s *AIService) chatJSON(ctx context.Context, systemPrompt, userPrompt string, target any) error {
	endpoint, err := url.JoinPath(s.baseURL, "v1", "chat", "completions")
	if err != nil {
		return err
	}
	body := map[string]any{
		"model": s.model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.4,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+s.apiKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("llm request failed: %s", response.Status)
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return err
	}
	if len(result.Choices) == 0 {
		return fmt.Errorf("llm response has no choices")
	}
	content := strings.TrimSpace(result.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("llm response content is empty")
	}
	return json.Unmarshal([]byte(content), target)
}

func parentDraft(issue, style, tone string) string {
	opening := map[string]string{
		"容易焦虑": "先回应您的担心，也谢谢您这么细致地关注孩子。",
		"比较敏感": "我会尽量具体地说明，避免让您只看到负面的部分。",
		"沟通积极": "谢谢您主动同步，这会让我们更快形成一致的支持方式。",
		"关注成绩": "我先把表现和后续提升路径分开说明，方便您判断重点。",
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
