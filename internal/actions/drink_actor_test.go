package actions

var (
	_ DrinkActor = (*UserActor)(nil)
	_ DrinkActor = (*MobActor)(nil)
)
