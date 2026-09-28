package actions

// DrinkActor is an Actor that can receive a potion's conditions at a scaled
// duration or an exact magnitude. It is its own interface, not two new Actor
// methods, because eight test fakes implement Actor and none of them drinks.
type DrinkActor interface {
	Actor
	AddConditionScaled(conditionId int, durationMult float64, source string)
	AddConditionMagnitude(conditionId int, triggers int, magnitude float64, source string)
}
