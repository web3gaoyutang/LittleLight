package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"nhooyr.io/websocket"
)

const (
	defaultXFASREndpoint             = "wss://iat-api.xfyun.cn/v2/iat"
	defaultXFASRMaxSessionSeconds    = 55
	defaultXFASRMaxConcurrentPerUser = 1
	defaultXFASRDailyLimitPerUser    = 80
	maxDictationAudioBase64Bytes     = 13000
)

type DictationOptions struct {
	AppID                string
	APIKey               string
	APISecret            string
	Endpoint             string
	MaxSessionSeconds    int
	MaxConcurrentPerUser int
	DailyLimitPerUser    int
}

type DictationService struct {
	options DictationOptions
	now     func() time.Time
}

type DictationSessionOptions struct {
	Language   string
	SampleRate int
}

type DictationClientEvent struct {
	Type       string `json:"type"`
	Language   string `json:"language,omitempty"`
	SampleRate int    `json:"sampleRate,omitempty"`
	Seq        int    `json:"seq,omitempty"`
	Audio      string `json:"audio,omitempty"`
}

type DictationServerEvent struct {
	Type         string `json:"type"`
	SessionID    string `json:"sessionId,omitempty"`
	SN           int    `json:"sn,omitempty"`
	Text         string `json:"text,omitempty"`
	StableText   string `json:"stableText,omitempty"`
	UnstableText string `json:"unstableText,omitempty"`
	Code         string `json:"code,omitempty"`
	Message      string `json:"message,omitempty"`
	DurationMS   int64  `json:"durationMs,omitempty"`
}

type xfIATRequest struct {
	Common   *xfIATCommon   `json:"common,omitempty"`
	Business *xfIATBusiness `json:"business,omitempty"`
	Data     xfIATData      `json:"data"`
}

type xfIATCommon struct {
	AppID string `json:"app_id"`
}

type xfIATBusiness struct {
	Language string `json:"language"`
	Domain   string `json:"domain"`
	Accent   string `json:"accent"`
	EOS      int    `json:"eos,omitempty"`
	DWA      string `json:"dwa,omitempty"`
	PTT      int    `json:"ptt,omitempty"`
	RLang    string `json:"rlang,omitempty"`
	NuNum    int    `json:"nunum,omitempty"`
}

type xfIATData struct {
	Status   int    `json:"status"`
	Format   string `json:"format,omitempty"`
	Encoding string `json:"encoding,omitempty"`
	Audio    string `json:"audio,omitempty"`
}

type xfIATResponse struct {
	SID     string       `json:"sid"`
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Data    *xfIATResult `json:"data"`
}

type xfIATResult struct {
	Status int       `json:"status"`
	Result xfIATText `json:"result"`
}

type xfIATText struct {
	SN  int         `json:"sn"`
	LS  bool        `json:"ls"`
	PGS string      `json:"pgs"`
	RG  []int       `json:"rg"`
	WS  []xfIATWord `json:"ws"`
}

type xfIATWord struct {
	BG int         `json:"bg"`
	CW []xfIATCell `json:"cw"`
}

type xfIATCell struct {
	W string `json:"w"`
}

type DictationAssembler struct {
	segments map[int]string
	order    []int
}

func NewDictationService(options DictationOptions) *DictationService {
	return &DictationService{
		options: normalizeDictationOptions(options),
		now:     time.Now,
	}
}

func normalizeDictationOptions(options DictationOptions) DictationOptions {
	options.Endpoint = strings.TrimSpace(options.Endpoint)
	if options.Endpoint == "" {
		options.Endpoint = defaultXFASREndpoint
	}
	if options.MaxSessionSeconds <= 0 || options.MaxSessionSeconds > 55 {
		options.MaxSessionSeconds = defaultXFASRMaxSessionSeconds
	}
	if options.MaxConcurrentPerUser <= 0 {
		options.MaxConcurrentPerUser = defaultXFASRMaxConcurrentPerUser
	}
	if options.DailyLimitPerUser <= 0 {
		options.DailyLimitPerUser = defaultXFASRDailyLimitPerUser
	}
	options.AppID = strings.TrimSpace(options.AppID)
	options.APIKey = strings.TrimSpace(options.APIKey)
	options.APISecret = strings.TrimSpace(options.APISecret)
	return options
}

func (s *DictationService) Configured() bool {
	return s != nil && s.options.AppID != "" && s.options.APIKey != "" && s.options.APISecret != ""
}

func (s *DictationService) Options() DictationOptions {
	if s == nil {
		return normalizeDictationOptions(DictationOptions{})
	}
	return s.options
}

func (s *DictationService) Run(ctx context.Context, client *websocket.Conn, initial DictationSessionOptions) error {
	if s == nil || !s.Configured() {
		return errors.New("dictation service is not configured")
	}
	sessionOptions, err := normalizeDictationSessionOptions(initial)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.options.MaxSessionSeconds)*time.Second)
	defer cancel()

	xfURL, err := BuildXFASRAuthURL(s.options.Endpoint, s.options.APIKey, s.options.APISecret, s.now())
	if err != nil {
		return err
	}
	xfConn, _, err := websocket.Dial(ctx, xfURL, nil)
	if err != nil {
		return fmt.Errorf("connect xfyun iat: %w", err)
	}
	defer xfConn.Close(websocket.StatusNormalClosure, "dictation session closed")

	sessionID := fmt.Sprintf("dict_%d", s.now().UnixNano())
	if err := writeWSJSON(ctx, client, DictationServerEvent{Type: "ready", SessionID: sessionID}); err != nil {
		return err
	}

	clientEvents := make(chan DictationClientEvent, 16)
	resultEvents := make(chan DictationServerEvent, 16)
	errs := make(chan error, 2)
	go readDictationClientEvents(ctx, client, clientEvents, errs)
	go s.readXFResults(ctx, xfConn, resultEvents, errs)

	startedAt := s.now()
	firstFrame := true
	stopped := false
	for {
		select {
		case event := <-resultEvents:
			if err := writeWSJSON(ctx, client, event); err != nil {
				return err
			}
			if event.Type == "final" {
				duration := s.now().Sub(startedAt).Milliseconds()
				_ = writeWSJSON(ctx, client, DictationServerEvent{Type: "done", Text: event.Text, DurationMS: duration})
				return nil
			}
		case err := <-errs:
			if errors.Is(err, errDictationClientClosed) {
				return nil
			}
			return err
		case event := <-clientEvents:
			switch event.Type {
			case "start":
				next, err := normalizeDictationSessionOptions(DictationSessionOptions{Language: event.Language, SampleRate: event.SampleRate})
				if err != nil {
					return err
				}
				sessionOptions = next
			case "audio":
				if stopped {
					continue
				}
				if len(event.Audio) > maxDictationAudioBase64Bytes {
					return fmt.Errorf("audio frame is too large")
				}
				status := 1
				var common *xfIATCommon
				var business *xfIATBusiness
				if firstFrame {
					status = 0
					common = &xfIATCommon{AppID: s.options.AppID}
					business = xfBusiness(sessionOptions)
					firstFrame = false
				}
				payload := xfIATRequest{
					Common:   common,
					Business: business,
					Data: xfIATData{
						Status:   status,
						Format:   xfAudioFormat(sessionOptions.SampleRate),
						Encoding: "raw",
						Audio:    event.Audio,
					},
				}
				if err := writeWSJSON(ctx, xfConn, payload); err != nil {
					return err
				}
			case "stop":
				stopped = true
				if err := writeWSJSON(ctx, xfConn, xfIATRequest{Data: xfIATData{Status: 2}}); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported dictation event type: %s", event.Type)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

var errDictationClientClosed = errors.New("dictation client closed")

func readDictationClientEvents(ctx context.Context, conn *websocket.Conn, events chan<- DictationClientEvent, errs chan<- error) {
	for {
		messageType, data, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway {
				errs <- errDictationClientClosed
				return
			}
			errs <- err
			return
		}
		if messageType != websocket.MessageText {
			continue
		}
		var event DictationClientEvent
		if err := json.Unmarshal(data, &event); err != nil {
			errs <- fmt.Errorf("invalid dictation client event: %w", err)
			return
		}
		events <- event
	}
}

func (s *DictationService) readXFResults(ctx context.Context, conn *websocket.Conn, resultEvents chan<- DictationServerEvent, errs chan<- error) {
	assembler := NewDictationAssembler()
	for {
		messageType, data, err := conn.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				errs <- ctx.Err()
				return
			}
			errs <- err
			return
		}
		if messageType != websocket.MessageText {
			continue
		}
		var response xfIATResponse
		if err := json.Unmarshal(data, &response); err != nil {
			errs <- fmt.Errorf("decode xfyun result: %w", err)
			return
		}
		if response.Code != 0 {
			errs <- fmt.Errorf("xfyun iat error %d: %s", response.Code, response.Message)
			return
		}
		if response.Data == nil {
			continue
		}
		event := EventFromXFResult(response)
		if event.Type == "" {
			continue
		}
		mergedText := assembler.ApplyXF(response.Data.Result)
		event.StableText = mergedText
		if event.Type == "final" {
			event.Text = mergedText
		}
		resultEvents <- event
	}
}

func EventFromXFResult(response xfIATResponse) DictationServerEvent {
	if response.Data == nil {
		return DictationServerEvent{}
	}
	result := response.Data.Result
	text := xfText(result.WS)
	eventType := "partial"
	if response.Data.Status == 2 || result.LS {
		eventType = "final"
	}
	return DictationServerEvent{
		Type:         eventType,
		SN:           result.SN,
		Text:         text,
		StableText:   text,
		UnstableText: "",
	}
}

func NewDictationAssembler() *DictationAssembler {
	return &DictationAssembler{segments: map[int]string{}}
}

func (a *DictationAssembler) Apply(sn int, text string, stableText string, unstableText string) string {
	if a == nil || sn <= 0 {
		return text
	}
	if _, ok := a.segments[sn]; !ok {
		a.order = append(a.order, sn)
	}
	a.segments[sn] = text
	return a.Text()
}

func (a *DictationAssembler) ApplyXF(result xfIATText) string {
	if a == nil {
		return ""
	}
	sn := result.SN
	text := xfText(result.WS)
	if sn <= 0 {
		return a.Text() + text
	}
	if result.PGS == "rpl" && len(result.RG) == 2 {
		start, end := result.RG[0], result.RG[1]
		for value := start; value <= end; value++ {
			delete(a.segments, value)
		}
		filtered := a.order[:0]
		for _, value := range a.order {
			if value < start || value > end {
				filtered = append(filtered, value)
			}
		}
		a.order = filtered
	}
	if _, ok := a.segments[sn]; !ok {
		a.order = append(a.order, sn)
	}
	a.segments[sn] = text
	return a.Text()
}

func (a *DictationAssembler) Text() string {
	if a == nil {
		return ""
	}
	var builder strings.Builder
	seen := map[int]bool{}
	for _, sn := range a.order {
		if seen[sn] {
			continue
		}
		seen[sn] = true
		builder.WriteString(a.segments[sn])
	}
	return builder.String()
}

func normalizeDictationSessionOptions(options DictationSessionOptions) (DictationSessionOptions, error) {
	language := strings.TrimSpace(options.Language)
	if language == "" {
		language = "zh_cn"
	}
	if language != "zh_cn" && language != "en_us" {
		return DictationSessionOptions{}, fmt.Errorf("language must be zh_cn or en_us")
	}
	sampleRate := options.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}
	if sampleRate != 16000 && sampleRate != 8000 {
		return DictationSessionOptions{}, fmt.Errorf("sampleRate must be 16000 or 8000")
	}
	return DictationSessionOptions{Language: language, SampleRate: sampleRate}, nil
}

func xfBusiness(options DictationSessionOptions) *xfIATBusiness {
	business := &xfIATBusiness{
		Language: options.Language,
		Domain:   "iat",
		Accent:   "mandarin",
		EOS:      3000,
		PTT:      1,
	}
	if options.Language == "zh_cn" {
		business.DWA = "wpgs"
		business.RLang = "zh-cn"
		business.NuNum = 1
	}
	return business
}

func xfAudioFormat(sampleRate int) string {
	if sampleRate == 8000 {
		return "audio/L16;rate=8000"
	}
	return "audio/L16;rate=16000"
}

func xfText(words []xfIATWord) string {
	var builder strings.Builder
	for _, word := range words {
		if len(word.CW) == 0 {
			continue
		}
		builder.WriteString(word.CW[0].W)
	}
	return builder.String()
}

func writeWSJSON(ctx context.Context, conn *websocket.Conn, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, data)
}

func BuildXFASRAuthURL(endpoint string, apiKey string, apiSecret string, now time.Time) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = defaultXFASREndpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	host := parsed.Host
	if host == "" {
		return "", fmt.Errorf("xfyun endpoint host is required")
	}
	date := now.UTC().Format(http.TimeFormat)
	requestLine := fmt.Sprintf("GET %s HTTP/1.1", parsed.EscapedPath())
	if requestLine == "GET  HTTP/1.1" {
		requestLine = "GET /v2/iat HTTP/1.1"
	}
	signatureOrigin := fmt.Sprintf("host: %s\ndate: %s\n%s", host, date, requestLine)
	mac := hmac.New(sha256.New, []byte(apiSecret))
	_, _ = mac.Write([]byte(signatureOrigin))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	authorizationOrigin := fmt.Sprintf(`api_key="%s", algorithm="hmac-sha256", headers="host date request-line", signature="%s"`, apiKey, signature)
	query := parsed.Query()
	query.Set("authorization", base64.StdEncoding.EncodeToString([]byte(authorizationOrigin)))
	query.Set("date", date)
	query.Set("host", host)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
