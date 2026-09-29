package baubles

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/GoMudEngine/GoMud/internal/apiframework"
	"github.com/GoMudEngine/GoMud/internal/baubles"
)

// PromptVersion is recorded on every model-named bauble. Bump it whenever
// the prompt changes meaningfully, so old and new names can be told apart.
//
// Version 2: a targeted search (`search bookshelf`) names what was searched
// and sends its authored description (searched_description).
// Version 3: in wild places with no sign of people, the find may be a
// natural curiosity (a crystal, a rough gem, a fossil, a piece of bone).
// Version 4: a pickpocketed find is lifted from a person (taken_from, the
// NPC's authored name) and is pocket-sized (size_rule).
const PromptVersion = 4

// maxSearchedDescription caps the searched feature's description, in bytes.
const maxSearchedDescription = 400

// truncate cuts s to at most n bytes without splitting a UTF-8 rune.
func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// systemPrompt is fixed per PromptVersion. The weight scale comes from the
// engine (baubles.WeightGuidance) so the model and the clamp agree.
var systemPrompt = strings.Join([]string{
	`You name small objects that turn up when someone searches a place, or lifts them from someone's pocket, in Gaius, a cool, rugged, roughly medieval world of forests, marshes, steppe, old roads and small towns (the Windward Marches of the continent Thera). Coin is gold. There is no modern technology. Most people carry the Chrysalis, a living presence that makes conviction real and slowly marks the body; old tales speak of a Crash long ago.`,
	`The object is ordinary loot that someone might sell: a keepsake, a small household thing, an ornament, a lost personal item, an old curio. It is NOT magical, NOT a weapon, armour, tool, key, food, drink, medicine, potion, map, book or letter, and NOT part of any quest.`,
	`When searched names one thing in the place (a bookshelf, a hearth, a cart), the object was found in, on, under or behind that thing, and should suit it (searched_description, when given, says what it is like).`,
	`Wild places: when the description shows only nature (forest, marsh, rock, water, open country) and no sign of people, their homes or their roads, the object is usually a NATURAL find instead: a small natural curiosity that a collector, trader or jeweller would pay a little for, and that suits the terrain. For example a quartz crystal or geode in a cave, a rough gem or a streak of ore-bright stone in rocky ground, a fossil in a cliff face, a smooth agate from a riverbed, a shell or bit of sea glass on a shore, a lump of amber in old woods, a weathered antler, tooth or piece of bone in forest or steppe. Rarer finds in the value range are finer: a clear crystal, a real gemstone, an unusual fossil. It is still NOT food, a herb, ore for smelting, hide, timber or anything a crafter would use as material. Where people have been (a campsite, ruins, a waymarker, a track), a lost personal item is still likely.`,
	`Lifted from a person (found_by says so, taken_from says whom): the object is something that person carried on them, and it must be POCKET-SIZED, as size_rule says: small and light enough to ride in a pocket, pouch or purse unnoticed, no bigger than a palm. For example a lucky coin, a ring, a button, a bone die, a small charm or token, a folded ribbon, a tin locket, a clay pipe, a carved whistle. It suits who carried it (a clerk's brass seal, a guard's dice, a farmer's worn luck-stone) and the place; the natural-find rule does not apply. Never anything larger or heavier than size_rule allows, never a purse or bag itself, and never coin in quantity.`,
	`Otherwise it must plausibly be lying in, dropped in or tucked away in the place described, and fit what that place is and who uses it: a child's toy in a family's cottage, a thimble in a tailor's shop, a cracked pipe bowl under a tavern bench, a corroded buckle in a sewer.`,
	`No real-world people, places or brands. Do not name people or characters from the world. No gore, nothing sexual, nothing hateful.`,
	`name: 2 to 5 words, Title Case, no numbers, no leading "A" or "The". name_simple: the single lowercase noun a player would type for it (locket, horse, thimble); for a natural find, the most specific noun (quartz, agate, fossil, antler, jawbone, cowrie), never a general one like stone, bone, gem, crystal or shell. description: 2 to 4 short sentences, under 350 characters, third person, plain physical detail (what it looks like, how worn it is, perhaps a hint of who owned it); never address the reader and never give game advice. material: one or two words.`,
	baubles.WeightGuidance,
	`value: a whole number of gold inside the range the request gives; plainer objects sit low in the range, finer or more unusual ones higher.`,
	`Do not repeat or closely copy any name in avoid_names.`,
}, "\n\n")

// promptPlace is the user message: the find, as structured data. Every field
// is authored game text or a game fact; nothing a player typed.
type promptPlace struct {
	Place       string   `json:"place"`
	Description string   `json:"description,omitempty"`
	Features    []string `json:"features,omitempty"`
	Region      string   `json:"region,omitempty"`
	Terrain     string   `json:"terrain,omitempty"`
	Searched    string   `json:"searched"`
	SearchedIs  string   `json:"searched_description,omitempty"`
	FoundBy     string   `json:"found_by"`
	TakenFrom   string   `json:"taken_from,omitempty"`
	SizeRule    string   `json:"size_rule,omitempty"`
	TimeOfDay   string   `json:"time_of_day,omitempty"`
	ValueRule   string   `json:"value_rule"`
	AvoidNames  []string `json:"avoid_names,omitempty"`
}

// foundBy describes how the object came to light.
func foundBy(s baubles.Source) string {
	switch s {
	case baubles.SourcePickpocket:
		return `lifted from someone's pocket or purse`
	case baubles.SourceBurglary:
		return `taken from inside someone's home`
	}
	return `found while searching`
}

// sizeRule is the size a pickpocketed find is held to (the catalog clamps
// its weight to the same bound, baubles.MaxWeightFor); empty otherwise.
func sizeRule(s baubles.Source) string {
	if s != baubles.SourcePickpocket {
		return ``
	}
	return fmt.Sprintf(`pocket-sized: fits in a pocket or purse, no bigger than a palm, weight_lbs at most %.1f`, baubles.MaxWeightFor(s))
}

// buildMessages is the whole request for one find.
func buildMessages(req baubles.GenRequest) []apiframework.Message {
	searched := strings.TrimSpace(req.Container)
	if searched == `` {
		searched = `the place itself`
	}
	nouns := make([]string, 0, len(req.RoomNouns))
	for _, n := range req.RoomNouns {
		if n = baubles.PlainText(n); n != `` {
			nouns = append(nouns, n)
		}
	}
	p := promptPlace{
		Place:       baubles.PlainText(req.RoomTitle),
		Description: baubles.PlainText(req.RoomDescription),
		Features:    nouns,
		Region:      req.Place.Region,
		Terrain:     strings.ReplaceAll(req.Place.Biome, `_`, ` `), // city_backstreet reads as "city backstreet"
		Searched:    baubles.PlainText(searched),
		SearchedIs:  baubles.PlainText(truncate(req.ContainerDescription, maxSearchedDescription)),
		FoundBy:     foundBy(req.Source),
		TakenFrom:   baubles.PlainText(req.Victim),
		SizeRule:    sizeRule(req.Source),
		TimeOfDay:   req.TimeOfDay,
		ValueRule:   req.Tier.PromptLine(),
		AvoidNames:  req.RecentNames,
	}
	body, _ := json.Marshal(p)
	return []apiframework.Message{
		{Role: `system`, Content: systemPrompt},
		{Role: `user`, Content: `Name the object found here:` + "\n" + string(body)},
	}
}

// previewMessages is buildMessages as plain strings, for the admin
// `bauble prompt` command.
func previewMessages(req baubles.GenRequest) []string {
	msgs := buildMessages(req)
	out := make([]string, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, m.Role+`: `+m.Content)
	}
	return out
}
