package baubles

import (
	"time"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// Status is where a record is in its life.
type Status string

// A record is only ever created once its text is final (Phase 4 generates
// BEFORE minting), so there is no "pending" status and no placeholder text.
const (
	StatusReady    Status = `ready`    // named by the model
	StatusFallback Status = `fallback` // a generic trinket (no API key, model off, or the call failed)
	StatusSold     Status = `sold`     // sold to a merchant; the item is gone
	StatusRetired  Status = `retired`  // text withdrawn by an admin; shows generic text
)

// Source is how the bauble came into the world.
type Source string

const (
	SourceSearch     Source = `search`
	SourcePickpocket Source = `pickpocket`
	SourceBurglary   Source = `burglary`
	SourceAdmin      Source = `admin`
)

// Generator is what wrote the record's text.
type Generator string

const (
	GeneratorOpenAI Generator = `openai`
	GeneratorLocal  Generator = `local`
	GeneratorAdmin  Generator = `admin`
)

// Record is one bauble: one unique object in the world. Records are never
// shared between items and never reused, so the provenance and theft fields
// describe exactly one object.
type Record struct {
	Id     string `yaml:"id"`
	Status Status `yaml:"status"`

	Name           string    `yaml:"name"`
	NameSimple     string    `yaml:"name_simple"`
	Description    string    `yaml:"description"`
	Material       string    `yaml:"material,omitempty"`
	Tier           ValueTier `yaml:"tier"`
	Value          int       `yaml:"value"`
	ValueProposed  int       `yaml:"value_proposed,omitempty"`
	WeightLbs      float64   `yaml:"weight_lbs"`
	WeightProposed float64   `yaml:"weight_proposed,omitempty"`

	// Provenance. Region is ZoneConfig.Region, or the zone when unset.
	Source        Source    `yaml:"source"`
	RoomId        int       `yaml:"room_id,omitempty"`
	Zone          string    `yaml:"zone,omitempty"`
	Region        string    `yaml:"region,omitempty"`
	Biome         string    `yaml:"biome,omitempty"`
	FoundByUserId int       `yaml:"found_by_user_id,omitempty"`
	FoundRound    uint64    `yaml:"found_round,omitempty"`
	FoundAt       time.Time `yaml:"found_at"`
	FoundIn       string    `yaml:"found_in,omitempty"`  // the feature searched (`search bookshelf`); empty for the room
	Household     bool      `yaml:"household,omitempty"` // left in the room because it belonged to the household there

	// Theft: set when a household's bauble is taken (MarkStolen), and by the
	// stealing work to come (Phase 6).
	Stolen         bool      `yaml:"stolen,omitempty"`
	StolenByUserId int       `yaml:"stolen_by_user_id,omitempty"`
	StolenFromRoom int       `yaml:"stolen_from_room,omitempty"`
	StolenFromMob  int       `yaml:"stolen_from_mob,omitempty"`
	StolenFromName string    `yaml:"stolen_from_name,omitempty"`
	StolenFaction  string    `yaml:"stolen_faction,omitempty"`
	StolenAt       time.Time `yaml:"stolen_at,omitempty"`
	StolenZone     string    `yaml:"stolen_zone,omitempty"` // the zone the theft happened in: heat is only there (HotIn)

	// After a theft (Phase 6c). A stolen bauble is hot for
	// BaubleStolenHeatHours after StolenAt, unless it was returned since
	// (Hot). RecognizedAt is when its owner last recognised it on someone;
	// once per theft. ReturnedAt is when it was last given back to its owner.
	// ReturnCredit* is the one time a return earned its thief reputation
	// (never set again, whoever steals it next).
	RecognizedAt         time.Time `yaml:"recognized_at,omitempty"`
	ReturnedAt           time.Time `yaml:"returned_at,omitempty"`
	ReturnCreditUserId   int       `yaml:"return_credit_user_id,omitempty"`
	ReturnCreditFactions []string  `yaml:"return_credit_factions,omitempty"`
	ReturnCreditAt       time.Time `yaml:"return_credit_at,omitempty"`
	ReturnCreditRound    uint64    `yaml:"return_credit_round,omitempty"` // the game round of that credit, to compare with crimes' rounds

	// GivenToMob is the mob id a player last gave the bauble to, when that
	// was not a return (MarkGiven). Picked from that mob's pocket, the
	// bauble is not the mark's own, so it does not become stolen goods.
	// A theft (MarkStolen) clears it.
	GivenToMob int `yaml:"given_to_mob,omitempty"`

	// Generation audit.
	Generator     Generator `yaml:"generator"`
	Model         string    `yaml:"model,omitempty"`
	PromptVersion int       `yaml:"prompt_version,omitempty"`
	Tokens        int       `yaml:"tokens,omitempty"`
	Moderated     bool      `yaml:"moderated,omitempty"`
	PlayerKey     bool      `yaml:"player_key,omitempty"` // named through the finder's own key
	EditedBy      string    `yaml:"edited_by,omitempty"`

	// VanishedAt is set when the bauble was left lying untaken for
	// BaubleUntakenHours and removed from the world (MarkVanished).
	VanishedAt time.Time `yaml:"vanished_at,omitempty"`

	SoldAt    time.Time `yaml:"sold_at,omitempty"`
	SoldValue int       `yaml:"sold_value,omitempty"`
}

// KeptToFinder reports whether the record's text is its finder's alone:
// text a player's own key wrote that was never moderated (owner ruling
// 2026-09-29). Derived, never stored, so a record named before the rule
// (PlayerKey, Moderated false) is kept to its finder too. Never promotable
// to the corpus either way (slice C takes only server-key moderated text).
func (r Record) KeptToFinder() bool {
	return r.PlayerKey && !r.Moderated
}

// Text shown for a bauble whose text an admin has withdrawn.
const (
	retiredName        = `Trinket`
	retiredDescription = `A small, worn trinket of no particular character. Someone might pay a little for it.`
)

// View is what the item layer shows for this record.
func (r Record) View() items.BaubleView {
	v := items.BaubleView{
		Name:        r.Name,
		NameSimple:  r.NameSimple,
		Description: r.Description,
		Value:       r.Value,
		WeightLbs:   r.WeightLbs,
	}
	if r.Status == StatusRetired {
		v.Name = retiredName
		v.NameSimple = `trinket`
		v.Description = retiredDescription
		return v
	}
	v.PlayerText = r.PlayerKey
	if r.KeptToFinder() {
		// Text a player's own key wrote that was never moderated (owner
		// ruling 2026-09-29): its finder reads it through the item layer's
		// viewer-aware accessors, everyone else the generic trinket.
		own := v
		v.Name, v.NameSimple, v.Description = genericName, genericNameSimple, genericDescriptionFor(r.Id)
		if r.FoundByUserId > 0 {
			v.FinderUserId, v.Finder = r.FoundByUserId, &own
		}
	}
	return v
}

// MaterialFor is the material as viewerUserId may read it (appraise): a
// finder-only record's is its finder's alone, like its name (View).
func (r Record) MaterialFor(viewerUserId int) string {
	if r.KeptToFinder() && (viewerUserId <= 0 || viewerUserId != r.FoundByUserId) {
		return ``
	}
	return r.Material
}
