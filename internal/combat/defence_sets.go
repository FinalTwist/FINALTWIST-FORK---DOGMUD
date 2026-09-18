package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// DefenceEntryOpts carries the situational filters the entry builder applies.
type DefenceEntryOpts struct {
	// ThirdPartyVsGrappler mirrors the melee-only filterDefensesForThirdParty
	// behaviour (a bystander swinging into a grapple faces a reduced set).
	ThirdPartyVsGrappler bool
}

// equipmentGatedMeleeDefences reproduces, branch for branch, the equipment
// gate that lived in characters.GetDefenseSequence (deleted by U6b Task 2 —
// this is its only surviving copy).
//
// ⚠️ PER-SLOT, NOT A MAIN-HAND LADDER. Until 2026-08-30 this was a four-branch
// ladder and every branch read Equipment.Weapon or Offhand, so an extra arm
// could not contribute a defence at all:
//
//   - IsUnarmedStyle() ran FIRST and reads the MAIN HAND ONLY, so claws in hand
//     one suppressed parry AND block across all six arms.
//   - IsDualWielding() reads Weapon+Offhand only and returned EARLY, so two
//     weapons hid a shield in arm three.
//   - the parry count was hardcoded at two, so arms three through six could
//     never add one.
//   - HasShield() was the only part that scanned every arm, and exactly one
//     branch could reach it.
//
// A player with the extra-arms mutation and a tower shield on their third arm
// was getting dodge and nothing else.
//
// The rule is now derived from what each arm actually HOLDS: one parry entry
// per parry-capable armed hand, plus block if ANY arm holds a shield. That
// reproduces every two-handed case the ladder produced — no weapon to dodge,
// weapon to parry, two weapons to parry+parry, weapon+shield to parry+block —
// and lets the extra-arms mutation do what it visibly promises.
//
// HasShield() still includes species NaturalBash, so an earth elemental blocks
// with no shield item; do NOT tighten it to BestBlockRating() > 0.
//
// ⚠️ INTENDED BEHAVIOUR CHANGE: an unarmed or claw fighter HOLDING A SHIELD now
// blocks, where the ladder gave them dodge alone. Two EMPTY hands are unchanged
// (no weapon, no shield, so dodge only), which is the build
// internal/skills/skills.go solves WeaponCombat 1.34 against.
func equipmentGatedMeleeDefences(c *characters.Character) []combatvocab.Defence {
	defenses := []combatvocab.Defence{combatvocab.DefenceDodge}

	for i := 0; i < c.ParryCapableArmCount(); i++ {
		defenses = append(defenses, combatvocab.DefenceParry)
	}

	if c.HasShield() {
		defenses = append(defenses, combatvocab.DefenceBlock)
	}

	return defenses
}

// thirdPartyGrappleDefences is the set-reduction half of the third-party
// grapple rule: an entangled defender attacked by a bystander keeps only
// block. filterDefensesForThirdParty (melee) layers the vulnerability
// messaging on top of this same rule.
func thirdPartyGrappleDefences(defSeq []combatvocab.Defence) []combatvocab.Defence {
	filtered := []combatvocab.Defence{}
	for _, def := range defSeq {
		if def == combatvocab.DefenceBlock {
			filtered = append(filtered, def)
		}
	}
	return filtered
}

// DefenceEntriesFor is THE defence-set NAME builder for every attack, melee
// included. It intersects combatvocab.EligibleDefences's table with the
// equipment gate copied verbatim from characters.GetDefenseSequence (which
// U6b Task 2 deleted):
//
//   - parry: wielded weapon AND !IsUnarmedStyle() — knuckle/claw fighters
//     never parry; appears TWICE when dual-wielding (two blades, two chances)
//   - block: wielded weapon AND HasShield() — which includes species
//     NaturalBash, so an earth elemental blocks with no shield item; do NOT
//     gate on BestBlockRating()
//   - dodge, quell and defy: always available on their pairings
//
// It returns NAMES ONLY. Scoring stays with the consumer: melee's candidate
// loop keeps its situational penalties/quoting/bookkeeping; the seam keeps
// GetDefenseScoreFor x defenceEffectiveness, and gains the prone penalties
// there (before U6b a prone defender dodged a bolt at full score while
// dodging a sword at penalty).
//
// THREE things a new defence must carry with it. It needs an arm in
// characters.GetDefenseScore, or it enters every contest at 0 and always
// loses. It needs a row in characters.DefensePool if it is not paid in
// stamina, or the pair charges the wrong pool.
//
// And it needs a row in DefenceSkillAndStat, which is the one whose absence
// fails SILENTLY and WIDELY. Without it that defence maps to ("", ""), so
// hooks.bestSwingDefence builds an award-nothing Candidate for it -- and if
// that candidate happens to roll highest, progression.BestOf reports false and
// the defender's ENTIRE ROUND trains nothing, the real dodge and parry
// candidates in the same slice included. Not a compile error, not a panic, and
// invisible in combat text. Unreachable today only because every row in
// combatvocab's eligibility table has a mapping.
func DefenceEntriesFor(shape combatvocab.Attack, defender *characters.Character, opts DefenceEntryOpts) []combatvocab.Defence {
	if defender == nil {
		return nil
	}

	set, ok := combatvocab.EligibleDefences(shape)
	if !ok {
		// A pair the table does not know. The constructors cannot build one
		// and the spell validator refuses one, so this is a struct literal
		// somewhere. Uncontested is what the old default arm did; the log is
		// what it did not.
		mudlog.Error("DefenceEntriesFor", "shape", shape.String(), "error", "attack pair is not in the eligibility table; resolving uncontested")
		return []combatvocab.Defence{}
	}

	gated := equipmentGatedMeleeDefences(defender)

	entries := []combatvocab.Defence{}
	for _, name := range set {
		switch name {
		case combatvocab.DefenceDodge, combatvocab.DefenceParry, combatvocab.DefenceBlock:
			// Physical defences: keep the gated multiplicity (dual-wield
			// contributes two parry entries).
			for i := 0; i < countDefenceName(gated, name); i++ {
				entries = append(entries, name)
			}
		default:
			// quell and defy are not equipment-gated.
			entries = append(entries, name)
		}
	}

	if opts.ThirdPartyVsGrappler {
		entries = thirdPartyGrappleDefences(entries)
	}

	return entries
}

func countDefenceName(set []combatvocab.Defence, name combatvocab.Defence) int {
	n := 0
	for _, s := range set {
		if s == name {
			n++
		}
	}
	return n
}
