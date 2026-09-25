package aicompanion

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// harmWorld is one lit room with her owner (user 1), a passer-by (user 2)
// and the companion herself (mob 42), PvP set as asked.
func harmWorld(t *testing.T, pvp string) (*users.UserRecord, *users.UserRecord, *rooms.Room, *mobs.Mob) {
	t.Helper()
	cfg := configs.GetConfig()
	cfg.GamePlay.PVP = configs.ConfigString(pvp)
	cfg.GamePlay.PVPMinimumSkillRanks = 0
	configs.SetConfigForTest(t, cfg)

	room := &rooms.Room{RoomId: 1, Zone: `HarmZone`, Title: `A Clearing`, Biome: `city`, Lamp: rooms.LampPtr(90)}
	t.Cleanup(rooms.SeedRoomsForTest(map[int]*rooms.Room{1: room},
		map[string]*rooms.ZoneConfig{`HarmZone`: {Name: `HarmZone`, RoomId: 1, RoomIds: map[int]struct{}{1: {}}}}))

	owner := users.NewTestUser(1, `corvin`, `Corvin`, 0)
	other := users.NewTestUser(2, `bram`, `Bram`, 0)
	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{1: owner, 2: other}))
	room.AddPlayer(1)
	room.AddPlayer(2)

	her := harmMob(t, room, 42, `Mara`)
	return owner, other, room, her
}

func harmMob(t *testing.T, room *rooms.Room, id int, name string) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{InstanceId: id, Character: characters.Character{
		Name: name, RoomId: room.RoomId, Health: 100, Conditions: conditions.New(),
	}}
	m.Character.HealthMax.Value = 100
	mobs.SetInstanceForTest(id, m)
	t.Cleanup(func() { mobs.SetInstanceForTest(id, nil) })
	room.AddMob(id)
	return m
}

// harmScene shows her the creature (t1) and the passer-by (t2).
func harmScene(mobId int, userId int) *scene {
	sc := &scene{RoomId: 1, byRef: map[string]*thing{}}
	sc.byRef[`t1`] = &thing{Ref: `t1`, Kind: `npc`, Name: `the caravan guard`, MobInstanceId: mobId}
	sc.byRef[`t2`] = &thing{Ref: `t2`, Kind: `player`, Name: `Bram`, UserId: userId}
	return sc
}

func TestHarmFollowsTheOwnersRules(t *testing.T) {
	owner, other, room, _ := harmWorld(t, configs.PVPDisabled)
	immune := harmMob(t, room, 300, `a caravan guard`)
	immune.PlayerAttackImmune = true
	quiet := harmMob(t, room, 301, `a quiet scribe`)
	quiet.NonCombatant = true
	wolf := harmMob(t, room, 302, `a grey wolf`)

	if ok, _ := harmAllowed(owner, room, immune.InstanceId, 0); ok {
		t.Fatal("her owner cannot attack a player_attack_immune creature, so she cannot either")
	}
	if ok, _ := harmAllowed(owner, room, quiet.InstanceId, 0); ok {
		t.Fatal("nor a non-combatant")
	}
	if ok, _ := harmAllowed(owner, room, wolf.InstanceId, 0); !ok {
		t.Fatal("a wolf is fair game for anyone")
	}
	if ok, _ := harmAllowed(owner, room, 0, other.UserId); ok {
		t.Fatal("with PvP off her owner could not fight Bram, so she will not")
	}
	if ok, _ := harmAllowed(owner, room, 0, owner.UserId); ok {
		t.Fatal("she never turns on her owner")
	}
	if ok, _ := harmAllowed(nil, room, wolf.InstanceId, 0); ok {
		t.Fatal("with nobody to answer for it, she starts nothing")
	}
}

func TestHarmRespectsPvpAndTheParty(t *testing.T) {
	owner, other, room, _ := harmWorld(t, configs.PVPEnabled)
	if ok, reason := harmAllowed(owner, room, 0, other.UserId); !ok {
		t.Fatalf("with PvP on, her owner could fight Bram: %s", reason)
	}
	p := parties.New(owner.UserId)
	p.UserIds = append(p.UserIds, other.UserId)
	t.Cleanup(p.Disband)
	if ok, _ := harmAllowed(owner, room, 0, other.UserId); ok {
		t.Fatal("but not someone in her owner's party")
	}
}

func TestAttackOnAProtectedCreatureIsRefused(t *testing.T) {
	owner, _, room, her := harmWorld(t, configs.PVPDisabled)
	immune := harmMob(t, room, 300, `a caravan guard`)
	immune.PlayerAttackImmune = true
	m, c, _ := strangerModule()

	out := m.performAction(c, her, owner, harmScene(immune.InstanceId, 2),
		ActionProposal{Verb: `attack`, Ref: `t1`}, []stimulus{{Kind: `heard`, FromOwner: true}}, 0, 0)
	if out.Issued || out.Refused == `` {
		t.Fatalf("her owner asking does not make a protected creature fair game: %+v", out)
	}

	immune.PlayerAttackImmune = false
	out = m.performAction(c, her, owner, harmScene(immune.InstanceId, 2),
		ActionProposal{Verb: `attack`, Ref: `t1`}, []stimulus{{Kind: `heard`, FromOwner: true}}, 0, 0)
	if !out.Issued {
		t.Fatalf("an unprotected one her owner points at is: %+v", out)
	}
}

func TestTheRefusalListIsStillHers(t *testing.T) {
	owner, _, room, her := harmWorld(t, configs.PVPDisabled)
	kid := harmMob(t, room, 300, `a frightened child`)
	m, c, _ := strangerModule()
	c.profile = &Profile{Name: `Mara`, Combat: CombatProfile{Refuse: []string{`child`}}}

	out := m.performAction(c, her, owner, harmScene(kid.InstanceId, 2),
		ActionProposal{Verb: `attack`, Ref: `t1`}, []stimulus{{Kind: `heard`, FromOwner: true}}, 0, 0)
	if out.Refused != `you will not raise a hand to them` {
		t.Fatalf("the engine would allow it, and she still will not: %+v", out)
	}
}

// harmSpells is one harmful and one helpful spell, both aimed at someone.
func harmSpells(t *testing.T, her *mobs.Mob) {
	t.Helper()
	t.Cleanup(spells.SeedSpellsForTest(map[string]*spells.SpellData{
		`burn`: {SpellId: `burn`, Name: `Burn`, EffectType: `damage`,
			AttackType: combatvocab.AttackSpell, DamageType: combatvocab.DamagePhysical, Targeting: combatvocab.TargetSingle},
		`mend`: {SpellId: `mend`, Name: `Mend`, EffectType: `heal`,
			AttackType: combatvocab.AttackNone, DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle},
	}))
	her.Character.SpellBook = map[string]int{`burn`: 1, `mend`: 1}
	her.Character.Conviction = 100
}

// spellRef is the ref she was shown for a spell.
func spellRef(t *testing.T, her *mobs.Mob, id string) string {
	t.Helper()
	for _, o := range spellsReady(her) {
		if o.Id == id {
			return o.Ref
		}
	}
	t.Fatalf("%s is not offered", id)
	return ``
}

func TestHarmfulCastsAreOwnerDrivenAndGated(t *testing.T) {
	owner, _, room, her := harmWorld(t, configs.PVPDisabled)
	harmSpells(t, her)
	wolf := harmMob(t, room, 302, `a grey wolf`)
	burn := spellRef(t, her, `burn`)
	m, c, _ := strangerModule()
	cast := func(to string, stims []stimulus) actionOutcome {
		return m.performAction(c, her, owner, harmScene(wolf.InstanceId, 2),
			ActionProposal{Verb: `cast`, Ref: burn, To: to}, stims, 0, 0)
	}

	if out := cast(`t1`, []stimulus{{Kind: `heard`, Speaker: `Bram`, AskerUserId: 2}}); out.Refused != `that is not a stranger's to ask for` {
		t.Fatalf("a stranger cannot have her burn anything: %+v", out)
	}
	if out := cast(`t1`, []stimulus{{Kind: `heard`, FromOwner: true}}); !out.Issued {
		t.Fatalf("her owner can, at a creature he could harm: %+v", out)
	}
	if out := cast(`t2`, []stimulus{{Kind: `heard`, FromOwner: true}}); out.Issued {
		t.Fatalf("but not at a person he could not fight: %+v", out)
	}
	if out := cast(`owner`, []stimulus{{Kind: `heard`, FromOwner: true}}); out.Issued {
		t.Fatalf("and never at him: %+v", out)
	}
	wolf.PlayerAttackImmune = true
	if out := cast(`t1`, []stimulus{{Kind: `heard`, FromOwner: true}}); out.Issued {
		t.Fatalf("nor at a creature players may not attack: %+v", out)
	}
}

func TestHelpfulCastsStayHerOwnJudgement(t *testing.T) {
	owner, _, _, her := harmWorld(t, configs.PVPDisabled)
	harmSpells(t, her)
	mend := spellRef(t, her, `mend`)
	m, c, _ := strangerModule()
	out := m.performAction(c, her, owner, harmScene(0, 2),
		ActionProposal{Verb: `cast`, Ref: mend, To: `owner`}, nil, 0, 0)
	if !out.Issued {
		t.Fatalf("mending her owner in a quiet moment needs nobody's word: %+v", out)
	}
}

func TestAreaHarmMustSpareEveryone(t *testing.T) {
	owner, _, room, her := harmWorld(t, configs.PVPDisabled)
	harmMob(t, room, 302, `a grey wolf`)
	if ok, _ := areaHarmAllowed(owner, room, her); ok {
		t.Fatal("Bram is in the room and her owner could not fight him")
	}
	room.RemovePlayer(2)
	if ok, reason := areaHarmAllowed(owner, room, her); !ok {
		t.Fatalf("with only her owner and a wolf, it is: %s", reason)
	}
	immune := harmMob(t, room, 300, `a caravan guard`)
	immune.PlayerAttackImmune = true
	if ok, _ := areaHarmAllowed(owner, room, her); ok {
		t.Fatal("a protected creature would be caught")
	}
}

func TestCombatTargetsFollowTheOwnersRules(t *testing.T) {
	owner, _, room, _ := harmWorld(t, configs.PVPDisabled)
	immune := harmMob(t, room, 300, `a caravan guard`)
	immune.PlayerAttackImmune = true
	wolf := harmMob(t, room, 302, `a grey wolf`)
	if mayStrike(owner, room, immune.InstanceId) {
		t.Fatal("the plan and her reflexes cannot turn her on a protected creature")
	}
	if !mayStrike(owner, room, wolf.InstanceId) {
		t.Fatal("but can on a wolf")
	}
	m, c, _ := strangerModule()
	c.fight = &fightState{Refs: map[string]int{`e1`: immune.InstanceId, `e2`: wolf.InstanceId}}
	m.applyCombatProposal(c, CombatProposal{Target: `e1`}, owner)
	if c.fight.TargetId != 0 {
		t.Fatal("the model choosing a protected creature is not taken")
	}
	m.applyCombatProposal(c, CombatProposal{Target: `e2`}, owner)
	if c.fight.TargetId != wolf.InstanceId {
		t.Fatal("choosing a wolf is")
	}
}
