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

## Gotchas

- **Every entry point is nil-safe.** With nothing installed each returns
  "not handled", so the engine behaves as before when the module is absent.
- **Callers hold the mud lock.** Both paths run on the game loop; the
  installed functions must never take `util.LockMud()` themselves.
- **No imports.** This package must stay dependency-free so anything can call
  it without creating an import cycle.

## Consumers

`internal/usercommands` (ask), `internal/hooks` (installs the respawner),
`modules/aicompanion` (installs the ask handler, calls the respawner).
