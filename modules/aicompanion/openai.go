package aicompanion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// chatMessage is one message in an OpenAI chat completions request. An
// assistant message may carry tool calls instead of content; a tool message
// answers one of them by id.
type chatMessage struct {
	Role       string
	Content    string
	ToolCalls  []toolCall
	ToolCallId string
}

// wireMessage is chatMessage as the API expects it: content is null on an
// assistant message that only calls tools.
type wireMessage struct {
	Role       string     `json:"role"`
	Content    *string    `json:"content"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallId string     `json:"tool_call_id,omitempty"`
}

func toWire(msgs []chatMessage) []wireMessage {
	out := make([]wireMessage, len(msgs))
	for i, m := range msgs {
		w := wireMessage{Role: m.Role, ToolCalls: m.ToolCalls, ToolCallId: m.ToolCallId}
		if !(m.Role == `assistant` && len(m.ToolCalls) > 0 && m.Content == ``) {
			content := m.Content
			w.Content = &content
		}
		out[i] = w
	}
	return out
}

// toolCall is a function call the model asks for.
type toolCall struct {
	Id       string       `json:"id"`
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// toolSpec offers the model one read-only question it may ask the game.
type toolSpec struct {
	Type     string  `json:"type"`
	Function toolDef `json:"function"`
}

type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
	Strict      bool           `json:"strict"`
}

type jsonSchemaFormat struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type responseFormat struct {
	Type       string           `json:"type"`
	JSONSchema jsonSchemaFormat `json:"json_schema"`
}

type chatRequest struct {
	Model               string         `json:"model"`
	Messages            []wireMessage  `json:"messages"`
	Tools               []toolSpec     `json:"tools,omitempty"`
	ToolChoice          string         `json:"tool_choice,omitempty"`
	ResponseFormat      responseFormat `json:"response_format"`
	MaxCompletionTokens int            `json:"max_completion_tokens,omitempty"`
	Temperature         *float64       `json:"temperature,omitempty"`
	ReasoningEffort     string         `json:"reasoning_effort,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content   string     `json:"content"`
			Refusal   string     `json:"refusal"`
			ToolCalls []toolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// modelCall is everything the background goroutine needs. It is built under
// the mud lock and then owned by the goroutine; it holds no game pointers.
type modelCall struct {
	BaseURL     string
	APIKey      string
	Model       string
	Timeout     time.Duration
	MaxTokens   int
	Temperature float64
	Messages    []chatMessage
	SchemaName  string
	Schema      map[string]any
	Effort      string // reasoning effort for reasoning models; empty = not sent
	Retry       bool   // retry once on a transient failure (429, 5xx, timeout)
	Tools       []toolSpec
	ToolChoice  string          // auto, none; empty = not sent
	Ctx         context.Context // cancelled when the answer can no longer be used
}

// modelResult is what comes back: the raw JSON content, which the caller
// parses against the schema it asked for. Err is set on any failure.
type modelResult struct {
	Content  string
	Tokens   int
	Latency  time.Duration
	Err      error
	Status   int  // HTTP status, 0 when the request never got an answer
	Canceled bool // the caller gave up on it; not the provider's fault

	// ToolCalls are the model's requests for more information, when it
	// asked instead of answering. ToolsUsed counts them across a decision.
	ToolCalls []toolCall
	ToolsUsed int

	// Filled by the decision path on the goroutine: the parsed decision
	// with any lines the moderation check flagged already removed.
	Parsed    *Decision
	ParseErr  error
	Moderated int // lines removed by moderation
}

var httpClient = &http.Client{}

// callModel performs a chat completions request, retrying once after a
// short pause when the failure looks transient and the call allows it.
func callModel(c modelCall) modelResult {
	res := callModelOnce(c)
	if c.Retry && transient(res) {
		time.Sleep(1500 * time.Millisecond)
		again := callModelOnce(c)
		again.Latency += res.Latency + 1500*time.Millisecond
		again.Tokens += res.Tokens
		return again
	}
	return res
}

// transient reports failures worth one retry: rate limits, server errors
// and requests that never got an answer.
func transient(r modelResult) bool {
	if r.Err == nil || r.Canceled {
		return false
	}
	return r.Status == 0 || r.Status == http.StatusTooManyRequests || r.Status >= 500
}

// callModelOnce performs one blocking chat completions request with the
// call's strict JSON schema. It must only ever run on a goroutine that does
// not hold the mud lock. The API key is sent in a header and never logged or
// returned.
func callModelOnce(c modelCall) modelResult {
	start := time.Now()
	res := modelResult{}

	req := chatRequest{
		Model:      c.Model,
		Messages:   toWire(c.Messages),
		Tools:      c.Tools,
		ToolChoice: c.ToolChoice,
		ResponseFormat: responseFormat{
			Type: `json_schema`,
			JSONSchema: jsonSchemaFormat{
				Name:   c.SchemaName,
				Strict: true,
				Schema: c.Schema,
			},
		},
		MaxCompletionTokens: c.MaxTokens,
	}
	if c.Temperature > 0 {
		t := c.Temperature
		req.Temperature = &t
	}
	req.ReasoningEffort = c.Effort

	body, err := json.Marshal(req)
	if err != nil {
		res.Err = err
		return res
	}

	parent := c.Ctx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, c.Timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+`/chat/completions`, bytes.NewReader(body))
	if err != nil {
		res.Err = err
		return res
	}
	httpReq.Header.Set(`Content-Type`, `application/json`)
	httpReq.Header.Set(`Authorization`, `Bearer `+c.APIKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		res.Err = err
		res.Latency = time.Since(start)
		res.Canceled = errors.Is(err, context.Canceled) || errors.Is(parent.Err(), context.Canceled)
		return res
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	res.Latency = time.Since(start)
	res.Status = resp.StatusCode
	if err != nil {
		res.Err = err
		return res
	}

	if resp.StatusCode != http.StatusOK {
		snippet := strings.TrimSpace(string(raw))
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		res.Err = fmt.Errorf(`model API status %d: %s`, resp.StatusCode, snippet)
		return res
	}

	var cr chatResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		res.Err = fmt.Errorf(`decode response: %w`, err)
		return res
	}
	res.Tokens = cr.Usage.TotalTokens

	if len(cr.Choices) == 0 {
		res.Err = errors.New(`model returned no choices`)
		return res
	}
	ch := cr.Choices[0]
	if ch.Message.Refusal != `` {
		res.Err = errors.New(`model refused`)
		return res
	}
	if len(ch.Message.ToolCalls) > 0 {
		res.ToolCalls = ch.Message.ToolCalls
		return res
	}
	if ch.FinishReason == `length` {
		res.Err = errors.New(`model output truncated (raise MaxCompletionTokens)`)
		return res
	}

	if strings.TrimSpace(ch.Message.Content) == `` {
		res.Err = errors.New(`model returned empty content`)
		return res
	}
	res.Content = ch.Message.Content
	return res
}

// moderationRequest and moderationResponse are the OpenAI moderations API.
type moderationRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type moderationResponse struct {
	Results []struct {
		Flagged bool `json:"flagged"`
	} `json:"results"`
}

// moderate checks texts with the moderation endpoint (F19.1) and returns
// which were flagged. On any error it returns nil: moderation failing must
// not silence the companion, and the in-character filters still apply.
func moderate(baseURL string, apiKey string, model string, timeout time.Duration, texts []string) ([]bool, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	body, err := json.Marshal(moderationRequest{Model: model, Input: texts})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+`/moderations`, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set(`Content-Type`, `application/json`)
	req.Header.Set(`Authorization`, `Bearer `+apiKey)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`moderation API status %d`, resp.StatusCode)
	}
	var mr moderationResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&mr); err != nil {
		return nil, err
	}
	if len(mr.Results) != len(texts) {
		return nil, fmt.Errorf(`moderation returned %d results for %d texts`, len(mr.Results), len(texts))
	}
	out := make([]bool, len(texts))
	for i, r := range mr.Results {
		out[i] = r.Flagged
	}
	return out, nil
}

// moderateDecision removes flagged speech (and a flagged sayto) from a
// decision. Runs on the model goroutine; touches no game state.
// moderateDecision checks what the companion means to say. strict is the
// policy for a check that could not be made: for speech a stranger
// prompted the lines are dropped (fail closed), and for the owner's own
// conversation they are kept (fail open), so a moderation outage costs a
// server its harassment cover rather than its companions.
func moderateDecision(d *Decision, baseURL string, apiKey string, model string, timeout time.Duration, strict bool) int {
	var texts []string
	for _, l := range d.Speech {
		texts = append(texts, l.Text)
	}
	hasSayto := d.Action.Verb == `sayto` && strings.TrimSpace(d.Action.Query) != ``
	if hasSayto {
		texts = append(texts, d.Action.Query)
	}
	flags, err := moderate(baseURL, apiKey, model, timeout, texts)
	if err != nil || flags == nil {
		if !strict {
			return 0
		}
		// The check failed and these words were not the owner's to prompt:
		// say nothing rather than say something unchecked.
		removed := len(d.Speech)
		d.Speech = nil
		if hasSayto {
			d.Action = ActionProposal{Verb: `none`}
			removed++
		}
		return removed
	}
	removed := 0
	kept := d.Speech[:0]
	for i, l := range d.Speech {
		if flags[i] {
			removed++
			continue
		}
		kept = append(kept, l)
	}
	d.Speech = kept
	if hasSayto && flags[len(flags)-1] {
		d.Action = ActionProposal{Verb: `none`}
		removed++
	}
	return removed
}

// listModels returns the model ids the key can use (GET /models), or nil
// when the list cannot be read.
func listModels(baseURL string, apiKey string) map[string]bool {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+`/models`, nil)
	if err != nil {
		return nil
	}
	req.Header.Set(`Authorization`, `Bearer `+apiKey)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var list struct {
		Data []struct {
			Id string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&list); err != nil {
		return nil
	}
	ids := make(map[string]bool, len(list.Data))
	for _, d := range list.Data {
		ids[d.Id] = true
	}
	return ids
}
