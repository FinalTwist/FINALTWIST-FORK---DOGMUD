# companionai Context

## Purpose

`internal/companionai` is the seam between the engine and
`modules/aicompanion`, the module that drives bonded AI companions.
`internal/` never imports `modules/`, so the engine calls the functions here
and the implementations are installed at boot.

## API

```go
type AskFunc func(userId int, mobInstanceId int, text string) bool
type RespawnFunc func(userId int, mobId int) int
type IdleFunc func(mobInstanceId int) bool
type RejoinFunc func(userId int) bool
type SnapshotFunc func(userId int) bool
type NpcAskFunc func(ownerUserId int, mobInstanceId int, text string, authorized bool) bool
type BondedFunc func(mobInstanceId int) bool
type HoldFunc func(userId int, mobInstanceId int) bool

func SetAskHandler(f AskFunc)
func RouteAsk(userId int, mobInstanceId int, text string) bool

func SetRespawner(f RespawnFunc)
func RespawnBonded(userId int, mobId int) int

func SetIdleHandler(f IdleFunc)
func RouteIdle(mobInstanceId int) bool

func SetRejoiner(f RejoinFunc)
func Rejoin(userId int) bool

func SetSnapshotter(f SnapshotFunc)
func Snapshot(userId int) bool

func SetNpcAsker(f NpcAskFunc)
func AskNpc(ownerUserId int, mobInstanceId int, text string, authorized bool) bool

func SetBondedCheck(f BondedFunc)
func IsBondedCompanion(mobInstanceId int) bool
func DrivesBonded() bool

func SetHolder(f HoldFunc)
func HoldPosition(userId int, mobInstanceId int) bool
```

- `RouteAsk` is called by `internal/usercommands/ask.go` before the normal
  companion and NPC paths. The aicompanion module installs the handler and
  claims asks aimed at companions it drives.
- `RouteIdle` is called by `internal/hooks/MobIdle_HandleIdleMobs.go` right
  after the sleeping check. The aicompanion module claims the idle tick of
  every companion it drives, so floor-loot grabs, behaviour-tree idle and
  canned idle emotes never compete with it.
- `Snapshot` is called by the aicompanion module every few rounds and after
  any trade or pickup, so the bonded companion's gear, gold and progression
  are in the owner's user record when the engine autosaves or shuts down.
  (The engine itself snapshots companions only at logout, so a restart
  would otherwise lose a session's purchases and loot.) `internal/hooks`
  installs `SnapshotBondedCompanion`.
- `Rejoin` is called by the aicompanion module when a companion that went
  off on an errand has no known way back to its owner. `internal/hooks`
  installs `RejoinBondedCompanions`, which reuses `TransportCompanions`, so
  the return looks exactly like following.
- `RespawnBonded` is called by the aicompanion module when a fallen bonded
  companion has recovered. `internal/hooks` installs the implementation
  (`RespawnBondedCompanion`), because it owns `applyCompanionState`.
- `HoldPosition` is called by `internal/hooks` `TransportCompanions` once
  per companion it is about to move, with that companion's mob instance.
  The aicompanion module installs `holdFollow`, which holds only the bonded
  companion it drives (its owner sneaking, or it walking in on foot a
  moment later); every other companion of the same owner follows as before.
- `DrivesBonded` is called by `internal/usercommands/dismiss.go`. It is true
  only while a bonded check is installed, which the aicompanion module does
  only when switched on; `dismiss` refuses a bonded companion while it is,
  and lets the owner part with one peacefully while it is not.

## Gotchas

- **Every entry point is nil-safe.** With nothing installed each returns
  "not handled", so the engine behaves as before when the module is absent.
- **Callers hold the mud lock.** Both paths run on the game loop; the
  installed functions must never take `util.LockMud()` themselves.
- **No imports.** This package must stay dependency-free so anything can call
  it without creating an import cycle.

## Consumers

`internal/usercommands` (ask, dismiss), `internal/hooks` (installs the
respawner, asks the follow hold), `internal/actions` and `internal/seeders`
(the bonded check), `modules/aicompanion` (installs the ask handler, the
holder and the bonded check, calls the respawner).
