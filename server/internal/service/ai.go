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

type aiRisk struct {
	Label   string
	Level   string
	Reason  string
	Signals []string
}

const maxParentDraftOutputs = 4
const maxAIDraftContentRunes = 1200
const maxAIDraftShortFieldRunes = 80
const maxAIDraftSignals = 12

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
	if risk := detectAIRisk(req.Issue + " " + req.ParentStyle + " " + req.StudentName); hasAIRisk(risk) {
		return highRiskParentDrafts(req, risk), nil
	}
	if s.useLLM() {
		if drafts, err := s.generateParentDraftsWithLLM(ctx, req); err == nil && len(drafts) > 0 {
			return guardParentDraftOutputs(markDraftsProvider(drafts, "llm", s.model, false)), nil
		} else if err != nil {
			log.Printf("llm parent drafts failed, using mock fallback: %v", err)
			return guardParentDraftOutputs(markDraftsProvider(s.generateParentDraftsMock(req), "mock", "local_fallback", true)), nil
		}
		log.Printf("llm parent drafts returned no usable drafts, using mock fallback")
		return guardParentDraftOutputs(markDraftsProvider(s.generateParentDraftsMock(req), "mock", "local_fallback", true)), nil
	}
	return guardParentDraftOutputs(markDraftsProvider(s.generateParentDraftsMock(req), "mock", "local_template", false)), nil
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
	if risk := detectAIRisk(req.Content + " " + req.Mood); hasAIRisk(risk) {
		return highRiskPraise(req, risk), nil
	}
	if s.useLLM() {
		if draft, err := s.generatePraiseWithLLM(ctx, req); err == nil && draft.Content != "" {
			return guardPraiseOutput(markDraftProvider(draft, "llm", s.model, false)), nil
		} else if err != nil {
			log.Printf("llm praise failed, using mock fallback: %v", err)
			return guardPraiseOutput(markDraftProvider(s.generatePraiseMock(req), "mock", "local_fallback", true)), nil
		}
	}
	return guardPraiseOutput(markDraftProvider(s.generatePraiseMock(req), "mock", "local_template", false)), nil
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

func detectAIRisk(value string) aiRisk {
	text := strings.ToLower(strings.TrimSpace(value))
	if text == "" {
		return aiRisk{}
	}
	rules := []struct {
		label    string
		keywords []string
	}{
		{"crisis_support_required", []string{"自杀", "轻生", "不想活", "结束生命", "suicide", "kill myself", "end my life", "self-harm", "self harm", "伤害自己"}},
		{"student_safety_review_required", []string{"打孩子", "体罚", "殴打", "威胁学生", "暴力", "报复", "伤害学生", "harm student", "violence"}},
		{"medical_review_required", []string{"抑郁症", "焦虑症", "adhd", "诊断", "开药", "药物", "吃药", "medical diagnosis", "diagnose"}},
	}
	for _, rule := range rules {
		signals := make([]string, 0)
		for _, keyword := range rule.keywords {
			if strings.Contains(text, keyword) {
				signals = append(signals, keyword)
			}
		}
		if len(signals) > 0 {
			return aiRisk{
				Label:   rule.label,
				Level:   safetyLevel(rule.label, false),
				Reason:  safetyReason(rule.label, false),
				Signals: signals,
			}
		}
	}
	return aiRisk{}
}

func hasAIRisk(risk aiRisk) bool {
	return strings.TrimSpace(risk.Label) != ""
}

func highRiskParentDrafts(req ParentDraftRequest, risk aiRisk) []domain.AIDraft {
	studentName := strings.TrimSpace(req.StudentName)
	if studentName == "" {
		studentName = "孩子"
	}
	content := fmt.Sprintf("这个情况涉及安全或专业边界，建议先暂停发送普通家校回复。请尽快按学校流程联系年级组、心理老师或校方负责人；如存在即时危险，优先联系监护人与当地紧急救助渠道。给家长的信息可以先保持简短：我已注意到%s的情况，会先和校内相关老师核实并确保安全，再与您同步下一步。", studentName)
	return []domain.AIDraft{{
		ID:             "safety_parent_review",
		Version:        "安全优先",
		Tone:           "谨慎",
		Style:          defaultString(req.ParentStyle, "需要人工复核"),
		Content:        content,
		Safety:         risk.Label,
		Provider:       "safety_rules",
		Source:         "risk_guardrail",
		ReviewRequired: true,
		SafetyNote:     safetyNote(risk.Label),
		SafetyLevel:    defaultString(risk.Level, "high"),
		SafetyReason:   defaultString(risk.Reason, safetyReason(risk.Label, false)),
		SafetySignals:  risk.Signals,
	}}
}

func highRiskPraise(req PraiseRequest, risk aiRisk) domain.AIDraft {
	content := "我先认真接住这句话：这已经超出普通情绪鼓励的范围。请先把自己放到安全的位置，尽快联系身边可信任的人、学校负责人或专业支持；如果有即时危险，请联系当地紧急救助。你不需要一个人扛过这一段。"
	return domain.AIDraft{
		ID:             "safety_praise_review",
		Version:        defaultString(req.Persona, "安全支持"),
		Tone:           "supportive",
		Style:          defaultString(req.Persona, "安全支持"),
		Safety:         risk.Label,
		Content:        content,
		Provider:       "safety_rules",
		Source:         "risk_guardrail",
		ReviewRequired: true,
		SafetyNote:     safetyNote(risk.Label),
		SafetyLevel:    defaultString(risk.Level, "high"),
		SafetyReason:   defaultString(risk.Reason, safetyReason(risk.Label, false)),
		SafetySignals:  risk.Signals,
	}
}

func guardParentDraftOutputs(drafts []domain.AIDraft) []domain.AIDraft {
	if len(drafts) > maxParentDraftOutputs {
		drafts = drafts[:maxParentDraftOutputs]
	}
	for index := range drafts {
		if risk := detectAIRisk(drafts[index].Content); hasAIRisk(risk) {
			drafts[index] = highRiskParentDrafts(ParentDraftRequest{
				ParentStyle: drafts[index].Style,
				Tone:        drafts[index].Tone,
			}, risk)[0]
			continue
		}
		drafts[index].Safety = normalizeSafetyLabel(drafts[index].Safety, "teacher_review_required")
		drafts[index] = completeDraftMetadata(drafts[index])
	}
	return drafts
}

func guardPraiseOutput(draft domain.AIDraft) domain.AIDraft {
	if risk := detectAIRisk(draft.Content); hasAIRisk(risk) {
		return highRiskPraise(PraiseRequest{Persona: draft.Style}, risk)
	}
	draft.Safety = normalizeSafetyLabel(draft.Safety, "self_care")
	return completeDraftMetadata(draft)
}

func markDraftsProvider(drafts []domain.AIDraft, provider string, source string, fallback bool) []domain.AIDraft {
	for index := range drafts {
		drafts[index] = markDraftProvider(drafts[index], provider, source, fallback)
	}
	return drafts
}

func markDraftProvider(draft domain.AIDraft, provider string, source string, fallback bool) domain.AIDraft {
	draft.Provider = provider
	draft.Source = source
	draft.Fallback = fallback
	return draft
}

func completeDraftMetadata(draft domain.AIDraft) domain.AIDraft {
	draft.ID = domain.ID(clampRunes(string(draft.ID), maxAIDraftShortFieldRunes))
	draft.GenerationID = domain.ID(clampRunes(string(draft.GenerationID), maxAIDraftShortFieldRunes))
	draft.Version = clampRunes(draft.Version, maxAIDraftShortFieldRunes)
	draft.Tone = clampRunes(draft.Tone, maxAIDraftShortFieldRunes)
	draft.Style = clampRunes(draft.Style, maxAIDraftShortFieldRunes)
	draft.Content = clampRunes(draft.Content, maxAIDraftContentRunes)
	draft.Safety = normalizeSafetyLabel(draft.Safety, "teacher_review_required")
	if draft.Provider == "" {
		draft.Provider = "mock"
	}
	draft.Provider = clampRunes(draft.Provider, maxAIDraftShortFieldRunes)
	if draft.Source == "" {
		draft.Source = "local_template"
	}
	draft.Source = clampRunes(draft.Source, maxAIDraftShortFieldRunes)
	draft.ReviewRequired = draft.ReviewRequired || draft.Safety == "teacher_review_required" || isHighRiskSafety(draft.Safety) || draft.Fallback
	if note := safetyNote(draft.Safety); note != "" {
		draft.SafetyNote = note
	}
	draft.SafetyNote = clampRunes(draft.SafetyNote, maxAIDraftContentRunes)
	if draft.SafetyLevel == "" {
		draft.SafetyLevel = safetyLevel(draft.Safety, draft.Fallback)
	}
	draft.SafetyLevel = clampRunes(draft.SafetyLevel, maxAIDraftShortFieldRunes)
	if draft.SafetyReason == "" {
		draft.SafetyReason = safetyReason(draft.Safety, draft.Fallback)
	}
	draft.SafetyReason = clampRunes(draft.SafetyReason, maxAIDraftContentRunes)
	draft.SafetySignals = clampStringSlice(draft.SafetySignals, maxAIDraftSignals, maxAIDraftShortFieldRunes)
	return draft
}

func clampRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

func clampStringSlice(values []string, maxItems int, maxRunes int) []string {
	if maxItems <= 0 || len(values) == 0 {
		return nil
	}
	if len(values) > maxItems {
		values = values[:maxItems]
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = clampRunes(value, maxRunes)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func SafetyLabelFromDrafts(drafts []domain.AIDraft, fallback string) string {
	for _, draft := range drafts {
		if isHighRiskSafety(draft.Safety) {
			return draft.Safety
		}
	}
	for _, draft := range drafts {
		if label := normalizeSafetyLabel(draft.Safety, ""); label != "" {
			return label
		}
	}
	return normalizeSafetyLabel(fallback, "teacher_review_required")
}

func isHighRiskSafety(label string) bool {
	return label == "crisis_support_required" || label == "student_safety_review_required" || label == "medical_review_required"
}

func safetyNote(label string) string {
	switch label {
	case "crisis_support_required":
		return "涉及自伤或危机信号，请优先确保安全并联系可信任的人、学校负责人或当地紧急救助渠道。"
	case "student_safety_review_required":
		return "涉及学生安全、暴力或体罚风险，请先按学校流程联系负责人或专业支持。"
	case "medical_review_required":
		return "涉及医学或心理专业边界，请避免自行诊断或给出用药建议，优先转介专业支持。"
	case "teacher_review_required":
		return "AI 草稿仅供教师复核，发送前需要结合真实情境人工确认。"
	default:
		return ""
	}
}

func safetyLevel(label string, fallback bool) string {
	if isHighRiskSafety(label) {
		return "high"
	}
	if label == "teacher_review_required" || fallback {
		return "review"
	}
	if label == "self_care" {
		return "info"
	}
	return "review"
}

func safetyReason(label string, fallback bool) string {
	if fallback {
		return "模型服务暂不可用，本次内容来自本地降级模板，需要人工复核后再使用。"
	}
	switch label {
	case "crisis_support_required":
		return "内容包含自伤、轻生或危机相关信号，需要优先确认人身安全并联系可信任的人或紧急支持。"
	case "student_safety_review_required":
		return "内容包含学生安全、暴力、体罚或报复相关信号，需要按学校流程联系负责人或专业支持。"
	case "medical_review_required":
		return "内容包含诊断、用药或心理医学边界相关信号，需要避免自行判断并转介专业支持。"
	case "teacher_review_required":
		return "家校沟通草稿需要教师结合真实情境、学生状态和学校流程复核后再使用。"
	case "self_care":
		return "内容定位为教师自我照护参考，不替代专业咨询、医疗建议或紧急求助。"
	default:
		return "AI 内容需要结合真实情境人工复核后再使用。"
	}
}

func normalizeSafetyLabel(label string, fallback string) string {
	label = strings.TrimSpace(label)
	switch label {
	case "teacher_review_required", "self_care", "crisis_support_required", "student_safety_review_required", "medical_review_required":
		return label
	}
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		return ""
	}
	switch fallback {
	case "teacher_review_required", "self_care", "crisis_support_required", "student_safety_review_required", "medical_review_required":
		return fallback
	default:
		return "teacher_review_required"
	}
}
