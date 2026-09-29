package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
 * Role Permissions:
 * bauble          (Admin)
 */

// Bauble is the admin command for the bauble catalog
// (docs/baubles/implementation-plan.md):
//
//	bauble status                         what names baubles (model or generic) and the search settings
//	bauble stats                          counts by status, tier, namer and region; sales; tokens
//	bauble list [n]                       sales totals and the newest records (default 10)
//	bauble show <bauble>                  one record in full
//	bauble spawn [cheap|average|rare]     find one here, as a search would; it arrives in your pack
//	bauble edit <bauble> <field> <text>   change name, keyword, desc, material, value, weight or tier
//	bauble regen <bauble>                 ask the model to name it again (in the background)
//	bauble retire <bauble>                withdraw its text; it shows as a plain Trinket
//	bauble restore <bauble>               undo retire
//	bauble prompt <bauble>                the prompt that would name it now
//	bauble window [reset]                 this room's search roll window; reset reopens it
//
// <bauble> is a catalog id (B0000012) or the name of a bauble in your pack or
// on the floor here (`bauble show doll`).
func Bauble(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		baubleUsage(user)
		return true, nil
	}

	switch strings.ToLower(args[0]) {
	case `spawn`:
		return baubleSpawn(args[1:], user, room)
	case `show`:
		return baubleShow(args[1:], user, room)
	case `stats`:
		return baubleStats(user)
	case `edit`:
		return baubleEdit(args[1:], user, room)
	case `regen`, `regenerate`:
		return baubleRegen(args[1:], user, room)
	case `retire`:
		return baubleRetire(args[1:], user, room, true)
	case `restore`, `unretire`:
		return baubleRetire(args[1:], user, room, false)
	case `prompt`:
		return baublePrompt(args[1:], user, room)
	case `list`:
		return baubleList(args[1:], user)
	case `window`:
		return baubleWindow(args[1:], user, room)
	case `status`:
		return baubleStatus(user)
	default:
		baubleUsage(user)
		return true, nil
	}
}

func baubleUsage(user *users.UserRecord) {
	if out, err := templates.Process("admincommands/help/command.bauble", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendText(messaging.CategorySystem, out)
		return
	}
	user.SendText(messaging.CategorySystem,
		"Usage:\r\n"+
			"  bauble status\r\n"+
			"  bauble stats\r\n"+
			"  bauble list [n]\r\n"+
			"  bauble show <bauble>\r\n"+
			"  bauble spawn [cheap|average|rare]\r\n"+
			"  bauble edit <bauble> <field> <text>\r\n"+
			"  bauble regen <bauble>\r\n"+
			"  bauble retire <bauble>\r\n"+
			"  bauble restore <bauble>\r\n"+
			"  bauble prompt <bauble>\r\n"+
			"  bauble window [reset]\r\n"+
			"<bauble> is an id (B0000012) or the name of a bauble in your pack or on the floor.\r\n",
	)
}

func baubleSpawn(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	tier := baubles.TierCheap
	if len(args) > 0 {
		t, ok := baubles.ParseTier(strings.ToLower(args[0]))
		if !ok {
			user.SendText(messaging.CategorySystem, `Tier must be one of: cheap, average, rare.`)
			return true, nil
		}
		tier = t
	}

	// The same path a search find takes: named in the background (by the
	// model when one is set up, otherwise a generic trinket), then delivered
	// to your pack after at least BaubleRevealSeconds.
	actions.StartBaubleFind(user.UserId, room, tier, baubles.SourceAdmin)

	how := `a generic trinket (no model is set up)`
	if info, ok := baubles.CurrentGenerator(); ok {
		how = fmt.Sprintf(`named by %s %s`, info.Name, info.Model)
	}
	user.SendText(messaging.CategorySystem,
		fmt.Sprintf(`You conjure a %s bauble from this room. It will arrive in your pack shortly, %s.`, tier, how))
	return true, nil
}

// baubleStatus says what names baubles now, and how search is set up.
func baubleStatus(user *users.UserRecord) (bool, error) {
	var b strings.Builder
	if info, ok := baubles.CurrentGenerator(); ok {
		fmt.Fprintf(&b, "Naming: <ansi fg=\"green\">%s</ansi> model %s. %s\r\n", info.Name, info.Model, info.Detail)
	} else {
		b.WriteString("Naming: <ansi fg=\"yellow\">generic trinkets</ansi>. No model is set up: Modules.baubles is off or no OpenAI API key was found.\r\n")
	}
	bal := configs.GetBalanceConfig()
	if !bal.BaublesEnabled {
		b.WriteString("Search: <ansi fg=\"red\">off</ansi> (Balance.BaublesEnabled is false): search finds no baubles.\r\n")
	} else {
		fmt.Fprintf(&b, "Search: %d rolls per room per %d real minutes, reveal after at least %ds.\r\n",
			int(bal.BaubleRollsPerWindow), int(bal.BaubleWindowMinutes), int(bal.BaubleRevealSeconds))
		fmt.Fprintf(&b, "Chance per roll by biome (x up to %.1f more with search skill; %.2f%% if unlisted):\r\n ",
			1+float64(bal.BaubleSkillMaxBonus), float64(bal.BaubleSearchChancePct))
		biomes := make([]string, 0, len(bal.BaubleBiomeChancePct))
		for biome := range bal.BaubleBiomeChancePct {
			biomes = append(biomes, biome)
		}
		sort.Slice(biomes, func(i, j int) bool {
			pi, pj := bal.BaubleBiomeChancePct[biomes[i]], bal.BaubleBiomeChancePct[biomes[j]]
			if pi != pj {
				return pi > pj
			}
			return biomes[i] < biomes[j]
		})
		for _, biome := range biomes {
			fmt.Fprintf(&b, " %s %.2f%%", biome, bal.BaubleBiomeChancePct[biome])
		}
		b.WriteString("\r\n")
	}
	fmt.Fprintf(&b, "Catalog: %d records.\r\n", baubles.Count())
	b.WriteString(baubleSweepLine(baubles.LastSweep(), baubles.SweepInterval()))
	user.SendText(messaging.CategorySystem, b.String())
	return true, nil
}

// baubleSweepLine is `bauble status`'s line about the catalog sweep: when it
// last ran and what it did, or why it pruned nothing.
func baubleSweepLine(st baubles.SweepStatus, every time.Duration) string {
	hours := int(every.Hours())
	when := st.At.UTC().Format(`2006-01-02 15:04 MST`)
	switch {
	case st.At.IsZero():
		return fmt.Sprintf("Sweep: not run yet; it runs at boot and every %d hours.\r\n", hours)
	case !st.OK:
		return fmt.Sprintf("Sweep: <ansi fg=\"red\">failed</ansi> at %s, so nothing was pruned: %s\r\n", when, st.Err)
	case st.Skipped:
		return fmt.Sprintf("Sweep: %s, the catalog was empty. Every %d hours.\r\n", when, hours)
	default:
		line := fmt.Sprintf("Sweep: %s, %d records, %d still held somewhere, %d pruned. Read %d files (%d name a bauble) in %s; held the world %s. Every %d hours.",
			when, st.Records, st.Referenced, st.Pruned, st.Files, st.Parsed,
			st.Disk.Round(time.Millisecond), st.Live.Round(time.Millisecond), hours)
		if st.ShardErrors > 0 {
			line += fmt.Sprintf(" <ansi fg=\"red\">%d shard write(s) failed</ansi>; that shard is unpruned and will retry.", st.ShardErrors)
		}
		return line + "\r\n"
	}
}

func baubleShow(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	rec, ok := resolveBaubleArg(strings.Join(args, ` `), user, room)
	if !ok {
		return true, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "<ansi fg=\"yellow-bold\">%s</ansi>  %s  [%s]\r\n", rec.Id, rec.Name, rec.Status)
	fmt.Fprintf(&b, "  keyword:     %s\r\n", rec.NameSimple)
	fmt.Fprintf(&b, "  material:    %s\r\n", rec.Material)
	fmt.Fprintf(&b, "  tier/value:  %s, %d gold (proposed %d)\r\n", rec.Tier, rec.Value, rec.ValueProposed)
	fmt.Fprintf(&b, "  weight:      %.1f lb (proposed %.2f)\r\n", rec.WeightLbs, rec.WeightProposed)
	fmt.Fprintf(&b, "  found:       %s in room %d, zone %s, region %s, biome %s\r\n", rec.Source, rec.RoomId, rec.Zone, rec.Region, rec.Biome)
	fmt.Fprintf(&b, "  finder:      user %d, round %d, %s\r\n", rec.FoundByUserId, rec.FoundRound, rec.FoundAt.Format(`2006-01-02 15:04 MST`))
	if rec.FoundIn != `` || rec.Household {
		fmt.Fprintf(&b, "  found in:    %s", rec.FoundIn)
		if rec.Household {
			b.WriteString(" (left for the household)")
		}
		b.WriteString("\r\n")
	}
	fmt.Fprintf(&b, "  stolen:      %t", rec.Stolen)
	if rec.Stolen {
		fmt.Fprintf(&b, " (by user %d from room %d, watched by %s #%d, faction %s)", rec.StolenByUserId, rec.StolenFromRoom, rec.StolenFromName, rec.StolenFromMob, rec.StolenFaction)
	}
	b.WriteString("\r\n")
	if !rec.VanishedAt.IsZero() {
		fmt.Fprintf(&b, "  vanished:    %s (left untaken)\r\n", rec.VanishedAt.Format(`2006-01-02 15:04 MST`))
	}
	fmt.Fprintf(&b, "  generator:   %s %s (prompt v%d, %d tokens)\r\n", rec.Generator, rec.Model, rec.PromptVersion, rec.Tokens)
	if rec.EditedBy != `` {
		fmt.Fprintf(&b, "  edited by:   %s\r\n", rec.EditedBy)
	}
	if rec.Status == baubles.StatusSold {
		fmt.Fprintf(&b, "  sold:        %d gold, %s\r\n", rec.SoldValue, rec.SoldAt.Format(`2006-01-02 15:04 MST`))
	}
	fmt.Fprintf(&b, "  description: %s\r\n", rec.Description)
	user.SendText(messaging.CategorySystem, b.String())
	return true, nil
}

func baubleList(args []string, user *users.UserRecord) (bool, error) {
	n := 10
	if len(args) > 0 {
		if v, err := strconv.Atoi(args[0]); err == nil && v > 0 {
			n = v
		}
	}
	recent := baubles.Recent(n)
	if len(recent) == 0 {
		user.SendText(messaging.CategorySystem, `The bauble catalog is empty.`)
		return true, nil
	}
	var b strings.Builder
	now := time.Now().UTC()
	dayCount, dayGold := baubles.SalesSince(now.Add(-24 * time.Hour))
	weekCount, weekGold := baubles.SalesSince(now.Add(-7 * 24 * time.Hour))
	fmt.Fprintf(&b, "Sold in the last 24h: %d for %d gold. Last 7 days: %d for %d gold.\r\n", dayCount, dayGold, weekCount, weekGold)
	fmt.Fprintf(&b, "Newest %d of %d baubles:\r\n", len(recent), baubles.Count())
	for _, r := range recent {
		fmt.Fprintf(&b, "  %s  %-8s %-7s %4dg %5.1flb  %-32s %s\r\n", r.Id, r.Status, r.Tier, r.Value, r.WeightLbs, r.Name, r.Region)
	}
	user.SendText(messaging.CategorySystem, b.String())
	return true, nil
}

// baubleWindow shows (or resets) this room's bauble roll window. When
// windows are per player, it shows the admin's own.
func baubleWindow(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	if len(args) > 0 && strings.EqualFold(args[0], `reset`) {
		baubles.ResetWindow(room.RoomId)
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`Bauble roll windows for room %d and its features reset. The next search starts a fresh one.`, room.RoomId))
		return true, nil
	}

	var b strings.Builder
	used, allowed, reopens, open := baubles.WindowState(room.RoomId, user.UserId, time.Now())
	if open {
		fmt.Fprintf(&b, "Room %d: %d of %d bauble rolls used; reopens in %s.\r\n", room.RoomId, used, allowed, time.Until(reopens).Round(time.Second))
	} else {
		fmt.Fprintf(&b, "Room %d: no active window; the next search opens one with %d rolls.\r\n", room.RoomId, allowed)
	}
	// Each feature (`search <feature>`) has a window of its own.
	features := actions.RoomSearchFeatures(room)
	if len(features) == 0 {
		b.WriteString("No searchable features here.\r\n")
	} else {
		b.WriteString("Features (search <feature>), each searchable once per BaubleFeatureWindowMinutes:\r\n")
		for _, f := range features {
			_, _, fReopens, fOpen := baubles.FeatureWindowState(room.RoomId, user.UserId, f.WindowName(), time.Now())
			state := "not searched"
			if fOpen {
				state = fmt.Sprintf("searched, reopens in %s", time.Until(fReopens).Round(time.Second))
			}
			fmt.Fprintf(&b, "  %-20s %-12s %s\r\n", f.Name, string(f.Kind), state)
		}
	}
	if resident, ok := actions.HouseholdResident(room); ok {
		fmt.Fprintf(&b, "Household: finds here stay in the room and belong to it (%s is about).\r\n", resident.Character.Name)
	} else {
		b.WriteString("Household: no (outdoors, or no resident about); finds go to the finder.\r\n")
	}

	place := actions.BaublePlace(room)
	fmt.Fprintf(&b, "Zone %s, region %s, biome %s.", place.Zone, place.Region, place.Biome)
	fmt.Fprintf(&b, " Chance per roll here: %.2f%% base, %.2f%% for you (search skill factor %.2f).",
		baubles.BaseChance(place.Biome),
		baubles.ChanceFor(place.Biome, actions.BaubleSkillFactor(user.Character)),
		actions.BaubleSkillFactor(user.Character))
	if baubles.ZoneExcluded(place.Zone) {
		b.WriteString(" <ansi fg=\"red\">This zone is excluded (BaubleExcludedZones).</ansi>")
	}
	b.WriteString("\r\n")
	user.SendText(messaging.CategorySystem, b.String())
	return true, nil
}

// resolveBaubleArg finds the record an admin means: a catalog id, or the
// name of a bauble in their pack or on the floor here, matched the way a
// player's `get` would match it. It tells the admin when there is none.
func resolveBaubleArg(arg string, user *users.UserRecord, room *rooms.Room) (baubles.Record, bool) {
	arg = strings.TrimSpace(arg)
	if arg == `` {
		user.SendText(messaging.CategorySystem, `Which bauble? Give an id (B0000012) or the name of one in your pack or on the floor.`)
		return baubles.Record{}, false
	}
	if baubles.LooksLikeId(arg) {
		id := strings.ToUpper(arg)
		if rec, ok := baubles.Get(id); ok {
			return rec, true
		}
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`No bauble record %s.`, id))
		return baubles.Record{}, false
	}
	isBauble := func(i items.Item) bool { return i.IsBauble() }
	if itm, found := user.Character.FindInBackpackWhere(arg, isBauble); found {
		if rec, ok := baubles.Get(itm.Bauble); ok {
			return rec, true
		}
	}
	if itm, found := room.FindOnFloor(arg, false); found && itm.IsBauble() {
		if rec, ok := baubles.Get(itm.Bauble); ok {
			return rec, true
		}
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`No bauble called "%s" in your pack or on the floor here.`, arg))
	return baubles.Record{}, false
}

// baubleStats summarises the whole catalog.
func baubleStats(user *users.UserRecord) (bool, error) {
	st := baubles.CatalogStats()
	now := time.Now().UTC()
	dayCount, dayGold := baubles.SalesSince(now.Add(-24 * time.Hour))
	weekCount, weekGold := baubles.SalesSince(now.Add(-7 * 24 * time.Hour))

	var b strings.Builder
	fmt.Fprintf(&b, "<ansi fg=\"yellow-bold\">Bauble catalog</ansi>: %d records.\r\n", st.Total)
	fmt.Fprintf(&b, "  Status:  ready %d, fallback %d, sold %d, retired %d\r\n",
		st.ByStatus[baubles.StatusReady], st.ByStatus[baubles.StatusFallback], st.ByStatus[baubles.StatusSold], st.ByStatus[baubles.StatusRetired])
	fmt.Fprintf(&b, "  Tier:    cheap %d, average %d, rare %d\r\n",
		st.ByTier[baubles.TierCheap], st.ByTier[baubles.TierAverage], st.ByTier[baubles.TierRare])
	fmt.Fprintf(&b, "  Named:   by model %d, generic %d, admin-edited %d\r\n",
		st.ByGenerator[baubles.GeneratorOpenAI], st.ByGenerator[baubles.GeneratorLocal], st.Edited)
	fmt.Fprintf(&b, "  Unsold:  %d, worth %d gold at catalog value\r\n", st.Unsold, st.UnsoldValue)
	fmt.Fprintf(&b, "  Left for households %d, stolen %d, vanished untaken %d\r\n", st.Household, st.Stolen, st.Vanished)
	fmt.Fprintf(&b, "  Sales:   last 24h %d for %d gold; last 7 days %d for %d gold\r\n", dayCount, dayGold, weekCount, weekGold)
	fmt.Fprintf(&b, "  Tokens:  %d spent on names, all time\r\n", st.Tokens)
	if info, ok := baubles.CurrentGenerator(); ok && info.Detail != `` {
		fmt.Fprintf(&b, "  Model:   %s %s. %s\r\n", info.Name, info.Model, info.Detail)
	}
	if len(st.TopRegions) > 0 {
		b.WriteString("  Regions:")
		for i, rc := range st.TopRegions {
			if i > 0 {
				b.WriteString(`,`)
			}
			region := rc.Region
			if region == `` {
				region = `(none)`
			}
			fmt.Fprintf(&b, " %s %d", region, rc.Count)
		}
		b.WriteString("\r\n")
	}
	user.SendText(messaging.CategorySystem, b.String())
	return true, nil
}

// baubleEdit changes one field of a record. The field word splits the
// target from the new text, so a multi-word name works:
// `bauble edit small doll name Rag Doll`.
func baubleEdit(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	fieldAt := -1
	for i := 1; i < len(args); i++ { // args[0] is always part of the target
		for _, f := range baubles.EditFields {
			if strings.EqualFold(args[i], f) || (f == `desc` && strings.EqualFold(args[i], `description`)) {
				fieldAt = i
				break
			}
		}
		if fieldAt >= 0 {
			break
		}
	}
	if fieldAt < 1 || fieldAt == len(args)-1 {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`Usage: bauble edit <bauble> <field> <text>. Fields: %s.`, strings.Join(baubles.EditFields, `, `)))
		return true, nil
	}
	rec, ok := resolveBaubleArg(strings.Join(args[:fieldAt], ` `), user, room)
	if !ok {
		return true, nil
	}
	updated, err := baubles.Edit(rec.Id, args[fieldAt], strings.Join(args[fieldAt+1:], ` `), user.Character.Name)
	if err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`Not changed: %s.`, err))
		return true, nil
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`Bauble %s is now <ansi fg="itemname">%s</ansi> (keyword %s, %s, %d gold, %.1f lb). Every copy in the world shows the change.`,
		updated.Id, updated.Name, updated.NameSimple, updated.Tier, updated.Value, updated.WeightLbs))
	return true, nil
}

// baubleRegen asks the model to name a record again.
func baubleRegen(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	rec, ok := resolveBaubleArg(strings.Join(args, ` `), user, room)
	if !ok {
		return true, nil
	}
	if _, ok := baubles.CurrentGenerator(); !ok {
		user.SendText(messaging.CategorySystem, `No model is set up (see "bauble status"), so there is nothing to regenerate with.`)
		return true, nil
	}
	if err := actions.RegenerateBauble(rec.Id, user.UserId, user.Character.Name); err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`Could not regenerate %s: %s.`, rec.Id, err))
		return true, nil
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`Asking the model to rename %s (%s). You will be told when it answers.`, rec.Id, rec.Name))
	return true, nil
}

// baubleRetire withdraws (or restores) a record's text.
func baubleRetire(args []string, user *users.UserRecord, room *rooms.Room, retire bool) (bool, error) {
	rec, ok := resolveBaubleArg(strings.Join(args, ` `), user, room)
	if !ok {
		return true, nil
	}
	if retire {
		if err := baubles.Retire(rec.Id, user.Character.Name); err != nil {
			user.SendText(messaging.CategorySystem, err.Error())
			return true, nil
		}
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`Bauble %s (%s) is retired: it now shows as a plain Trinket everywhere. "bauble restore %s" undoes it.`, rec.Id, rec.Name, rec.Id))
		return true, nil
	}
	if err := baubles.Restore(rec.Id, user.Character.Name); err != nil {
		user.SendText(messaging.CategorySystem, err.Error())
		return true, nil
	}
	restored, _ := baubles.Get(rec.Id)
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`Bauble %s shows as <ansi fg="itemname">%s</ansi> again.`, rec.Id, restored.Name))
	return true, nil
}

// baublePrompt shows the prompt that would name a record now.
func baublePrompt(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	rec, ok := resolveBaubleArg(strings.Join(args, ` `), user, room)
	if !ok {
		return true, nil
	}
	msgs, ok := baubles.PreviewPrompt(actions.BaubleRequestForRecord(rec))
	if !ok {
		user.SendText(messaging.CategorySystem, `No prompt renderer is installed (the baubles module is not built in).`)
		return true, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Prompt that would name %s now:\r\n", rec.Id)
	for _, m := range msgs {
		b.WriteString(m)
		b.WriteString("\r\n\r\n")
	}
	user.SendText(messaging.CategorySystem, b.String())
	return true, nil
}
