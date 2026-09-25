package aicompanion

import (
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// End-of-session reflection (F1.6, F3.8, F3.12). When the owner leaves, the
// companion looks back over the session in private: a short account of it,
// at most two higher-level conclusions, and any facts about the owner it
// missed at the time. Nothing here is spoken; it only shapes what the
// companion remembers and believes next time.

// Reflection is the model's reply to a reflection request.
type Reflection struct {
	Summary     string   `json:"summary"`
	Conclusions []string `json:"conclusions"`
	Facts       []string `json:"facts"`
	Mood        string   `json:"mood"`
}

func reflectionSchema() map[string]any {
	return object(
		[]string{`summary`, `conclusions`, `facts`, `mood`},
		map[string]any{
			`summary`: str(`Two or three sentences, in your own voice, about what happened today and how it went between you. Empty if nothing happened worth recalling.`),
			`conclusions`: map[string]any{
				`type`:        `array`,
				`description`: `Zero to two broader conclusions you draw about them, yourself or the two of you. Only real insights.`,
				`items`:       map[string]any{`type`: `string`},
			},
			`facts`: map[string]any{
				`type`:        `array`,
				`description`: `Facts about them you learned today and have not already noted. Usually empty.`,
				`items`:       map[string]any{`type`: `string`},
			},
			`mood`: map[string]any{`type`: `string`, `enum`: moods},
		},
	)
}

// reflectionInput is the plain data a reflection is built from.
type reflectionInput struct {
	Profile     *Profile
	OwnerName   string
	Opinion     Opinion
	Facts       []Fact
	Reflections []Memory
	Lines       []Line
	Session     int
}

func buildReflectionMessages(in reflectionInput) []chatMessage {
	p := in.Profile
	var sys strings.Builder
	fmt.Fprintf(&sys, "You are %s, a person in the world of Gaius. %s\n", p.Name, strings.TrimSpace(p.Summary))
	writeList(&sys, `Personality`, p.Personality)
	fmt.Fprintf(&sys, "Your travelling companion %s has gone to rest. Alone, you think back over the time you just spent together.\n", in.OwnerName)
	sys.WriteString("Think in character. Be honest and specific; do not invent events that did not happen. You know nothing of computers, games or artificial minds.\n")

	var usr strings.Builder
	for _, w := range opinionWords(in.Opinion, in.OwnerName) {
		usr.WriteString(w)
		usr.WriteString("\n")
	}
	if len(in.Facts) > 0 {
		fmt.Fprintf(&usr, "\nWhat you already know about %s:\n", in.OwnerName)
		for _, f := range in.Facts {
			fmt.Fprintf(&usr, "- %s\n", f.Text)
		}
	}
	if len(in.Reflections) > 0 {
		usr.WriteString("\nConclusions you had already drawn:\n")
		for _, r := range in.Reflections {
			fmt.Fprintf(&usr, "- %s\n", r.Text)
		}
	}
	usr.WriteString("\nWhat happened this time (oldest first; quoted text is exactly what was said or done):\n")
	for _, l := range in.Lines {
		usr.WriteString(formatLine(l, p.Name))
		usr.WriteString("\n")
	}
	usr.WriteString("\nReflect.")
	return []chatMessage{
		{Role: `system`, Content: sys.String()},
		{Role: `user`, Content: usr.String()},
	}
}

// startReflection launches a background reflection for a mind whose owner
// has just left. Called under the mud lock from detach. The mind pointer
// stays in the module's cache, so if the owner returns before the reply
// lands, both the new session and the reflection work on the same Mind.
func (m *AICompanionModule) startReflection(mind *Mind, p *Profile, ownerName string, sessionStart int64) {
	// Consent covers everything that leaves the server, not only what is
	// said in the moment: a player who declined must not have their session
	// posted to OpenAI the instant they log out.
	if !m.consented(mind.OwnerUserId) {
		return
	}
	if !m.cfg.ReflectOnLogout || !m.modelReady(mind.OwnerUserId) {
		return
	}
	lines := mind.linesSince(sessionStart)
	if len(lines) < m.cfg.MinSessionLinesForReflection {
		return
	}
	if len(lines) > 80 {
		lines = lines[len(lines)-80:]
	}

	in := reflectionInput{
		Profile:     p,
		OwnerName:   ownerName,
		Opinion:     mind.Opinion,
		Facts:       mind.Facts,
		Reflections: mind.reflections(3),
		Lines:       append([]Line(nil), lines...),
		Session:     mind.SessionCount,
	}
	ts := m.settingsFor(tierDeep, false)
	if ts.Model == `` {
		return
	}
	call := modelCall{
		BaseURL:     m.cfg.BaseURL,
		APIKey:      m.apiKey(),
		Model:       ts.Model,
		Timeout:     ts.Timeout,
		MaxTokens:   ts.MaxTokens,
		Temperature: m.cfg.Temperature,
		Messages:    buildReflectionMessages(in),
		SchemaName:  `companion_reflection`,
		Schema:      reflectionSchema(),
		Effort:      ts.Effort,
		Retry:       m.cfg.RetryTransient,
		OwnerUserId: mind.OwnerUserId,
	}
	reserved := worstCaseTokens(estimateTokens(call.Messages), ts.MaxTokens, 0, call.Retry)
	if !m.tryReserveTokens(mind.OwnerUserId, reserved) {
		return // the day's thinking is spent; the session simply goes unrecorded
	}
	m.callsToday++
	key := mindIdentifier(mind.OwnerUserId, mind.MobId)
	session := mind.SessionCount

	go func() {
		defer func() {
			if r := recover(); r != nil {
				mudlog.Error(`aicompanion`, `action`, `reflection`, `panic`, r, `stack`, string(debug.Stack()))
			}
		}()

		res := m.callModel(call)

		util.LockMud()
		defer util.UnlockMud()
		m.applyReflection(key, call.OwnerUserId, session, call.Model, reserved, res)
	}()
}

// applyReflection stores a reflection. Runs under the mud lock.
func (m *AICompanionModule) applyReflection(key string, ownerId int, session int, model string, reserved int, res modelResult) {
	m.rollDay()
	m.recordCall(tierDeep, res)
	m.breakerResult(res.Err, time.Now())
	if mind := m.minds[key]; mind == nil {
		// Nobody left to remember it, but the reservation still has to go
		// back, to the owner it was held against.
		m.settleTokens(ownerId, reserved, res.Tokens)
	}
	if modelRefused(res) {
		m.models.refuse(model)
	}

	mind := m.minds[key]
	if mind == nil {
		return
	}
	m.settleTokens(mind.OwnerUserId, reserved, res.Tokens)
	mind.TokensLifetime += int64(res.Tokens)
	if res.Err != nil {
		m.logModelError(res.Err)
		return
	}
	var r Reflection
	if err := parseJSONContent(res.Content, &r); err != nil {
		m.logModelError(fmt.Errorf(`parse reflection: %w`, err))
		return
	}

	now := time.Now().Unix()
	if text := cleanText(r.Summary, 600); text != `` && !breaksCharacter(text) {
		mind.addSummary(Summary{Unix: now, Session: session, Text: text}, m.cfg.MaxSummaries)
	}
	for i, c := range r.Conclusions {
		if i >= 2 {
			break
		}
		if text := cleanText(c, maxRememberRunes); text != `` && !breaksCharacter(text) {
			mind.addMemory(Memory{Unix: now, Kind: `reflection`, Text: text, Importance: 7, Emotion: `neutral`}, m.cfg.MaxMemories)
		}
	}
	for i, f := range r.Facts {
		if i >= 3 {
			break
		}
		if text := cleanText(f, maxFactRunes); text != `` && !breaksCharacter(text) {
			// These come out of her own thinking afterwards, not from
			// anything the owner said in the moment, so they are held as
			// her reading of things and never as "they told me".
			mind.addFact(Fact{Unix: now, Text: text, Source: `inferred`, Confidence: `low`}, m.cfg.MaxFacts)
		}
	}
	if mood := strings.ToLower(strings.TrimSpace(r.Mood)); isMood(mood) {
		mind.Mood = mood
		mind.MoodSetUnix = now
	}

	if err := saveMind(m.plug, mind); err != nil {
		mudlog.Error(`aicompanion`, `action`, `saveMind`, `owner`, mind.OwnerUserId, `error`, err)
	}
	if c := m.ctrls[mind.OwnerUserId]; c != nil && c.mind == mind {
		c.dirty = false
	}
	if m.cfg.LogDecisions {
		mudlog.Info(`aicompanion`, `action`, `reflection`, `owner`, mind.OwnerUserId, `session`, session,
			`conclusions`, len(r.Conclusions), `facts`, len(r.Facts), `tokens`, res.Tokens)
	}
}
