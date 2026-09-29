package apiframework

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// HTTPClient carries every request to a provider. A variable so tests can
// swap the transport.
var HTTPClient = &http.Client{}

// Endpoint is where a request goes and the key it carries: the server's own
// (Server()), for every HTTP request. The key goes only in the
// Authorization header and is never logged or returned.
type Endpoint struct {
	BaseURL string
	APIKey  string
}

// Carries says what a request carries. The zero value is the guarded kind,
// so a request nobody thought to classify is treated as carrying a player's
// words.
type Carries int

const (
	// CarriesPlayerData is anything built from a player's words or doings
	// (every AI companion prompt, and the moderation of what she means to
	// say). It needs an Admit hook, or it is refused.
	CarriesPlayerData Carries = iota
	// CarriesNoPlayerData is authored game text and nothing of any player's
	// (bauble naming, the model list).
	CarriesNoPlayerData
)

// Admit is the caller's consent door, run immediately before a request
// leaves (path is the API path, for the caller's own records). An error
// stops the request; nothing is sent.
type Admit func(path string) error

// ErrNotAdmitted is a request carrying player data sent without an Admit
// hook. It never left: the framework fails closed.
var ErrNotAdmitted = errors.New(`not sent: a request carrying player data needs a consent check`)

// Exchange is one request's outcome.
type Exchange struct {
	Status  int    // HTTP status; 0 when no answer came back
	Raw     []byte // the reply body, at most 1 MiB
	Sent    bool   // the request may have reached the provider, so may be billed
	Latency time.Duration
	Err     error
}

// admitted runs the door for a request of this kind.
func admitted(carries Carries, admit Admit, path string) error {
	if admit != nil {
		if err := admit(path); err != nil {
			return err
		}
		return nil
	}
	if carries != CarriesNoPlayerData {
		return ErrNotAdmitted
	}
	return nil
}

// Post sends body to ep.BaseURL+path with ep's key, through the door, and
// returns the reply. It blocks: call it only off the mud lock. A context
// already done when it is called sends nothing.
func Post(ctx context.Context, ep Endpoint, path string, body []byte, carries Carries, admit Admit) Exchange {
	start := time.Now()
	out := Exchange{}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		out.Err = err
		return out
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		out.Err = err
		return out
	}
	req.Header.Set(`Content-Type`, `application/json`)
	req.Header.Set(`Authorization`, `Bearer `+ep.APIKey)
	if err := admitted(carries, admit, req.URL.Path); err != nil {
		out.Err = err
		return out
	}

	resp, err := HTTPClient.Do(req)
	out.Latency = time.Since(start)
	// It left unless the connection was never made.
	out.Sent = err == nil || !NeverConnected(err)
	if err != nil {
		out.Err = err
		return out
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	out.Latency = time.Since(start)
	out.Status = resp.StatusCode
	out.Raw = raw
	out.Err = err
	return out
}

// NeverConnected reports an HTTP failure that happened before any byte of
// the request could have left: the connection was never made.
func NeverConnected(err error) bool {
	var op *net.OpError
	return errors.As(err, &op) && op.Op == `dial`
}

type moderationRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type moderationResponse struct {
	Results []struct {
		Flagged bool `json:"flagged"`
	} `json:"results"`
}

// Moderate checks texts with the moderation endpoint and returns which were
// flagged, one per text. On any error it returns the error and no flags:
// what to do then is the caller's policy (fail closed or open).
func Moderate(ep Endpoint, model string, timeout time.Duration, texts []string, carries Carries, admit Admit) ([]bool, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	body, err := json.Marshal(moderationRequest{Model: model, Input: texts})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ex := Post(ctx, ep, `/moderations`, body, carries, admit)
	if ex.Err != nil {
		return nil, ex.Err
	}
	if ex.Status != http.StatusOK {
		return nil, fmt.Errorf(`moderation API status %d`, ex.Status)
	}
	var mr moderationResponse
	if err := json.Unmarshal(ex.Raw, &mr); err != nil {
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

// ListModels returns the model ids the key can use (GET /models), or nil
// when the list cannot be read. It carries the key and nothing of any
// player's.
func ListModels(ep Endpoint) map[string]bool {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep.BaseURL+`/models`, nil)
	if err != nil {
		return nil
	}
	req.Header.Set(`Authorization`, `Bearer `+ep.APIKey)
	resp, err := HTTPClient.Do(req)
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

// String prints an Endpoint with its key redacted, so no %v, %+v or %s of
// an Endpoint, or of anything holding one (ServerSettings), in a log line
// or a test failure message can ever show a key.
func (e Endpoint) String() string {
	key := ``
	if e.APIKey != `` {
		key = `[redacted]`
	}
	return `{BaseURL:` + e.BaseURL + ` APIKey:` + key + `}`
}

// GoString is String for %#v.
func (e Endpoint) GoString() string { return `apiframework.Endpoint` + e.String() }
