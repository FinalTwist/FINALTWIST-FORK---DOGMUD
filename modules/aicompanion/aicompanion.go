// Package aicompanion drives bonded AI companions: persistent companions
// (characters.CompanionBonded) that talk, remember and form opinions through
// a language model, while acting in the world only through ordinary mob
// commands. See context.md and docs/aicompanion/.
package aicompanion

import (
	"embed"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/companionai"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// helpFiles are the player-facing help pages. They are mounted with the
// commands they describe, so a server with the module off has no help for a
// feature it does not have.
//
//go:embed files/datafiles/*
var helpFiles embed.FS

//go:embed primer.txt
var primerText string

// maxPendingStimuli bounds how many unanswered things a companion holds on
// to while a call is in flight or it is rate limited. Older ones drop off.
const maxPendingStimuli = 6

// controller is the runtime half of one bonded companion: the link between
// its owner, its mob instance and its mind. Controllers exist only while the
// owner is online. Every field is read and written under the mud lock.
type controller struct {
	ownerUserId int
	profile     *Profile
	mind        *Mind

	instanceId   int    // live mob instance, 0 while fallen or not yet spawned
	fellRound    uint64 // round the fall was noticed, 0 when standing
	greeted      bool   // session greeting already queued this session
	farewellSaid bool   // goodbye already queued for the owner's current quit

	sessionStartUnix    int64         // when this session began (for reflection)
	lastSocialUnix      int64         // last time anyone spoke or acted socially near it
	lastInitiativeCheck int64         // last time a quiet-spell roll was made
	lastAttackBy        map[int]int64 // attacker user id -> last reaction time

	traces     []traceEntry  // recent decisions, for the admin trace view
	lastPrompt []chatMessage // last request sent, for the admin prompt view
	lastIntent string        // what it meant to do last time
	recentPath []string      // rooms it has just walked through, newest last

	worldRev     uint64        // bumped whenever what it can see changes materially
	cancelCall   func()        // cancels the model call in flight, if any
	convo        *conversation // talk in progress, gathered into one memory at its end
	lastGold     int           // purse as last seen, to spot coin it did not earn
	lastGoldSeen int           // 1 once the purse has been read at least once
	lastSnapshot uint64        // round its gear and gold were last copied to the owner's record
	snapshotDue  bool          // something changed its gear or gold; snapshot next round

	leaveAt      uint64 // round it walks away when nothing is left to stay for
	leaveAskedAt int64  // when it asked to part ways, waiting on the owner
	partAskedAt  int64  // when the owner last typed companion-part
	leaveWhy     string

	lastRoomId    int             // room the companion was in last round
	seenKeys      map[string]bool // notable scene things already noticed here
	lastNotice    int64           // last "noticed" stimulus
	lastAutonomy  int64           // last "idle" stimulus
	lastIdleEmote int64           // last local idle emote
	lastPastime   int64           // last time it found something to do with itself
	lastGearUp    int64           // last time it sorted its gear out
	lastSneakTry  uint64          // round it last tried to slip into the shadows
	followPending bool            // a follow step is on its way and not yet taken
	followSince   uint64          // the round that step was issued

	askAuth         *askAuthority  // the owner's leave to put one question to one NPC
	ownerWasDown    bool           // the owner was on the ground when last looked at
	knownConditions map[int]bool   // what ailed them both when last looked at
	lastAilment     int64          // last time it remarked on one
	lastSavedSeen   int64          // when the mind was last written to disk
	budgetSpent     bool           // its last decision went unpaid for want of allowance
	pendingAct      *pendingAction // issued command awaiting its outcome
	travel          *travelPlan    // trip in progress
	apartSince      uint64         // round it was first found apart from its owner, 0 when together

	inFlightDirect  bool   // the call in flight answers someone speaking to it
	thinkShown      bool   // a thinking gesture was already made for this call
	partyKey        string // owner's party members last seen, sorted names
	partyKnown      bool   // partyKey has been read at least once this session
	ownerTalkedAway int64  // last time the owner spoke to someone else

	fight         *fightState       // fight in progress
	agenda        []int             // goal ids chosen for this session
	skillBase     map[string]string // favoured skill ranks when last checked
	lastGoalCheck uint64            // round goals were last checked

	seq      uint64 // bumps on every dispatch; stale results are dropped
	inFlight bool
	lastCall time.Time
	pending  []stimulus

	paused  bool
	dirty   bool
	lastErr string
}

func (c *controller) push(s stimulus) {
	if c.paused {
		return
	}
	c.pending = append(c.pending, s)
	if len(c.pending) > maxPendingStimuli {
		c.pending = append([]stimulus(nil), c.pending[len(c.pending)-maxPendingStimuli:]...)
	}
}

// AICompanionModule is the module singleton.
type AICompanionModule struct {
	plug     *plugins.Plugin
	cfg      Config
	profiles map[string]*Profile
	byMob    map[int]*Profile
	ctrls    map[int]*controller // keyed by owner user id
	minds    map[string]*Mind    // every mind loaded since boot, by mindIdentifier

	budgetDay   string
	tokensToday int
	callsToday  int
	errorsToday int
	lastErrLog  time.Time

	bonds             bondState             // who has met or turned away a companion
	pendingMeet       map[int]*meetWait     // characters waiting to meet one
	meetingPlace      map[int]string        // where each first meeting happened
	models            modelChooser          // automatic model choice per tier
	stats             map[string]*tierStats // per model tier, since boot
	ownerTokens       map[int]int           // tokens today per companion owner
	strangerTokens    map[int]int           // tokens today spent on behalf of a passer-by
	breakerUntil      time.Time             // model calls paused until then
	consecutiveErrors int
	outstanding       int // tokens held for calls that have not come back
	lastBudgetLog     time.Time
}

var module AICompanionModule

func init() {
	module = AICompanionModule{
		plug:     plugins.New(`aicompanion`, `0.1.0`),
		profiles: map[string]*Profile{},
		byMob:    map[int]*Profile{},
		ctrls:    map[int]*controller{},
		minds:    map[string]*Mind{},

		pendingMeet:  map[int]*meetWait{},
		meetingPlace: map[int]string{},
	}
	// No data-overlays/config.yaml is shipped. A plugin overlay is pushed
	// into the live config AFTER _datafiles/config.yaml is read, and
	// overwrites it (only config-overrides.yaml is protected), so an
	// overlay would quietly ignore the file an operator edits. Every
	// default lives in buildConfig instead, and is documented in
	// docs/aicompanion/settings.md and in the config.yaml block.

	// Admin only, and registered whether or not the module is switched on,
	// so an operator can always ask it why nothing is happening.
	module.plug.AddUserCommand(`aicompanion`, module.cmdAICompanion, true, true)

	module.plug.Callbacks.SetOnLoad(module.onLoad)
	module.plug.Callbacks.SetOnSave(module.onSave)
}

func (m *AICompanionModule) onLoad() {
	m.cfg = loadConfig(m.plug)

	// Switched off, the module stops here: nothing is loaded, no listener
	// is registered and no seam is installed, so the engine's nil checks
	// find nothing and the server behaves as though this module were not
	// built at all.
	if !m.cfg.Enabled {
		mudlog.Info(`aicompanion`, `enabled`, false,
			`message`, `switched off; set Modules.aicompanion.Enabled: true to use it`)
		return
	}

	profiles, errs := loadProfiles()
	for _, err := range errs {
		mudlog.Error(`aicompanion`, `action`, `loadProfiles`, `error`, err)
	}
	for id, p := range profiles {
		if mobs.GetMobSpec(mobs.MobId(p.MobId)) == nil {
			mudlog.Error(`aicompanion`, `action`, `loadProfiles`, `profile`, id,
				`error`, fmt.Sprintf(`mob template %d does not exist; profile disabled`, p.MobId))
			continue
		}
		m.profiles[id] = p
		m.byMob[p.MobId] = p
	}

	// The player commands, the companion's own mob commands and the help
	// files exist only while the module is on, so a server that leaves it
	// off has no dead commands in its registry, no help pages for a feature
	// that is not there, and nothing advertised to web clients.
	m.registerCommands()

	m.loadBonds()
	m.loadBudget()

	events.RegisterListener(events.NewRound{}, m.onNewRound)
	events.RegisterListener(events.CharacterCreated{}, m.onCharacterCreated)
	events.RegisterListener(events.PlayerDespawn{}, m.onPlayerDespawn)
	events.RegisterListener(events.Communication{}, m.onCommunication)
	events.RegisterListener(events.Emote{}, m.onEmote)
	events.RegisterListener(events.GiftAccepted{}, m.onGiftAccepted)
	events.RegisterListener(events.PlayerAttackedMob{}, m.onPlayerAttackedMob)
	events.RegisterListener(events.Healed{}, m.onHealed)
	events.RegisterListener(events.MobDeath{}, m.onMobDeath)
	events.RegisterListener(events.PlayerDeath{}, m.onPlayerDeath)
	companionai.SetAskHandler(m.handleAsk)
	companionai.SetIdleHandler(m.handleIdle)
	companionai.SetHolder(m.holdFollow)
	companionai.SetBondedCheck(m.isBonded)

	if m.cfg.RejectedBaseURL != `` {
		mudlog.Error(`aicompanion`, `action`, `config`, `error`,
			`BaseURL `+m.cfg.RejectedBaseURL+` is not an OpenAI endpoint over https; set AllowCustomEndpoint to use another provider. Using the official endpoint.`)
	}
	mudlog.Info(`aicompanion`, `enabled`, m.cfg.Enabled, `profiles`, len(m.profiles),
		`model`, m.cfg.Model, `apiKeyPresent`, m.apiKey() != ``)
	m.probeModels()
}

// onSave persists every changed mind. The plugin layer routes the write
// through the durable autosave queue.
func (m *AICompanionModule) onSave() error {
	if !m.cfg.Enabled {
		return nil
	}
	m.saveBudget()
	var firstErr error
	now := time.Now().Unix()
	for _, c := range m.ctrls {
		c.mind.LastSeenUnix = now
		// "When I last saw you" changes every minute and is what her
		// greeting is built on, so it is written even when nothing else
		// happened; otherwise a crash leaves her greeting the owner as
		// though no time had passed since the last thing she said.
		if !c.dirty && now-c.lastSavedSeen < 300 {
			continue
		}
		if err := saveMind(m.plug, c.mind); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue // try again next autosave rather than sit on the change
		}
		c.lastSavedSeen = now
		c.dirty = false
	}
	return firstErr
}

// getMind returns the one Mind for an owner and profile, loading it on first
// use. Minds stay cached for the life of the process, so a reflection that
// lands after its owner logged back in updates the same Mind the new
// session is using instead of a stale copy.
func (m *AICompanionModule) getMind(ownerUserId int, p *Profile) *Mind {
	key := mindIdentifier(ownerUserId, p.MobId)
	if mind, ok := m.minds[key]; ok {
		return mind
	}
	mind := loadMind(m.plug, ownerUserId, p)
	m.minds[key] = mind
	return mind
}

// apiKey is the OpenAI key: the environment variable named by APIKeyEnv
// (OPENAI_API_KEY by default, the variable OpenAI's own tools use), or, for a
// single-player server, APIKey in the config file. It is never logged.
func (m *AICompanionModule) apiKey() string {
	if k := strings.TrimSpace(os.Getenv(m.cfg.APIKeyEnv)); k != `` {
		return k
	}
	return m.cfg.APIKey
}

// rollDay resets the daily counters at the UTC date boundary.
func (m *AICompanionModule) rollDay() {
	day := time.Now().UTC().Format(`2006-01-02`)
	if day != m.budgetDay {
		m.budgetDay = day
		// Calls still in flight keep their reservation across the rollover,
		// or settling them afterwards would credit back tokens the new day
		// never spent.
		m.tokensToday = m.outstanding
		m.callsToday = 0
		m.errorsToday = 0
		m.ownerTokens = map[int]int{}
		m.strangerTokens = map[int]int{}
	}
}

// modelReady reports whether a model call may be made right now: a model
// and key are set, the circuit breaker is closed, and neither the server's
// nor this owner's companion's daily budget is spent. ownerId 0 skips the
// per-companion check.
func (m *AICompanionModule) modelReady(ownerId ...int) bool {
	if !m.cfg.Enabled || m.apiKey() == `` {
		return false
	}
	if m.breakerOpen(time.Now()) {
		return false
	}
	m.rollDay()
	if m.cfg.DailyTokenBudget > 0 && m.tokensToday >= m.cfg.DailyTokenBudget {
		return false
	}
	if len(ownerId) > 0 && ownerId[0] > 0 && !m.ownerBudgetLeft(ownerId[0]) {
		return false
	}
	return true
}

// bondedCompanionOf returns the owner's bonded companion record that has a
// loaded profile, or nil.
func (m *AICompanionModule) bondedCompanionOf(u *users.UserRecord) (*characters.CompanionInfo, *Profile) {
	if u == nil || u.Character == nil {
		return nil, nil
	}
	for i := range u.Character.Companions {
		c := &u.Character.Companions[i]
		if c.SourceType != characters.CompanionBonded {
			continue
		}
		if p, ok := m.byMob[c.MobId]; ok {
			return c, p
		}
	}
	return nil, nil
}

// controllerForInstance finds the controller driving a mob instance.
func (m *AICompanionModule) controllerForInstance(instanceId int) *controller {
	if instanceId <= 0 {
		return nil
	}
	for _, c := range m.ctrls {
		if c.instanceId == instanceId {
			return c
		}
	}
	return nil
}

// sortedOwnerIds gives a stable iteration order for status output.
func (m *AICompanionModule) sortedOwnerIds() []int {
	ids := make([]int, 0, len(m.ctrls))
	for id := range m.ctrls {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func (m *AICompanionModule) logModelError(err error) {
	m.errorsToday++
	// One line per ten seconds at most; an outage must not flood the log.
	if time.Since(m.lastErrLog) < 10*time.Second {
		return
	}
	m.lastErrLog = time.Now()
	mudlog.Warn(`aicompanion`, `action`, `modelCall`, `error`, err.Error(), `errorsToday`, m.errorsToday)
}

// bumpWorld marks that what the companion can see has changed materially:
// it moved, a fight started or ended, it fell, or it was respawned. A model
// reply built on the old picture is dropped rather than acted on.
func (c *controller) bumpWorld() {
	c.worldRev++
}

// cancelInFlight stops a model call whose answer can no longer be used, so
// it does not run on and spend tokens. The call's token reservation is
// settled by the goroutine itself, whichever way it ends.
func (c *controller) cancelInFlight() {
	if c.cancelCall != nil {
		c.cancelCall()
		c.cancelCall = nil
	}
	c.inFlight = false
}

// isBonded reports whether a mob instance is a bonded companion this module
// drives. Used by the engine to leave its own per-template opinion score
// alone for companions, which keep their feelings in their own mind.
func (m *AICompanionModule) isBonded(mobInstanceId int) bool {
	if !m.cfg.Enabled {
		return false
	}
	return m.controllerForInstance(mobInstanceId) != nil
}

// registerCommands puts the player and companion commands into the live
// registries. Called from onLoad, after the config is read, so nothing is
// registered on a server that never switches the module on.
// usercommands.RegisterCommand is used directly rather than the plugin
// helper because only it can say a command is allowed in combat.
func (m *AICompanionModule) registerCommands() {
	usercommands.RegisterCommand(`companion-unstick`, m.cmdUnstick, false, true, false)
	usercommands.RegisterCommand(`companion-part`, m.cmdPart, false, true, false)
	usercommands.RegisterCommand(`companion-court`, m.cmdCourt, false, false, false)
	usercommands.RegisterCommand(`companion-boundary`, m.cmdBoundary, false, true, false)
	usercommands.RegisterCommand(`companion-ask`, m.cmdAskFor, false, false, false)

	// mobcommands.RegisterCommand, not plug.AddMobCommand: the plugin
	// helper only fills a map that plugins.Load copies into the registry
	// before onLoad runs, so a command added here would never arrive and
	// the companion would be told it does not know how to do it.
	mobcommands.RegisterCommand(cmdCompanionLoot, mobCompanionLoot, false)
	mobcommands.RegisterCommand(cmdCompanionTakeout, mobCompanionTakeout, false)
	mobcommands.RegisterCommand(cmdCompanionUnlock, mobCompanionUnlock, false)
	mobcommands.RegisterCommand(cmdCompanionBuy, mobCompanionBuy, false)
	mobcommands.RegisterCommand(cmdCompanionFollow, mobCompanionFollow, false)

	// The help pages are mounted with the commands they describe.
	if err := m.plug.AttachFileSystem(helpFiles); err != nil {
		mudlog.Error(`aicompanion`, `action`, `attachHelp`, `error`, err)
	}
}
