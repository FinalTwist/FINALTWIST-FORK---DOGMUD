package configs

// Lighting is every knob the graded light model reads, in one small struct.
//
// 🔑 This exists for a measured reason. GetBalanceConfig() copies a 424-field
// struct under two read locks and benchmarks at 99.75 ns, against 8.23 ns for
// the small GetTimingConfig(). Fifteen call sites were paying that copy to read
// a single int, and Room.LightLevel() is called from per-round loops. The knobs
// stay declared on Balance, so the yaml schema is unchanged; only the read path
// is narrowed.
type Lighting struct {
	BlindBelow            int
	DimBelow              int
	ExitsAbove            int
	DefaultVisionStrength int

	DoublingStep  float64
	WorldLatitude float64
	EquinoxNoon   float64
	Starlight     float64
	MoonsFull     float64

	MoonWeightSwiftmoon float64
	MoonWeightWanderer  float64
	MoonWeightEye       float64
}

// GetLightingConfig returns the lighting knobs without copying Balance.
func GetLightingConfig() Lighting {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()

	b := &configData.Balance
	return Lighting{
		BlindBelow:            int(b.LightBlindBelow),
		DimBelow:              int(b.LightDimBelow),
		ExitsAbove:            int(b.LightExitsAbove),
		DefaultVisionStrength: int(b.LightDefaultVisionStrength),

		DoublingStep:  float64(b.LightDoublingStep),
		WorldLatitude: float64(b.WorldLatitude),
		EquinoxNoon:   float64(b.LightEquinoxNoon),
		Starlight:     float64(b.LightStarlight),
		MoonsFull:     float64(b.LightMoonsFull),

		MoonWeightSwiftmoon: float64(b.LightMoonWeightSwiftmoon),
		MoonWeightWanderer:  float64(b.LightMoonWeightWanderer),
		MoonWeightEye:       float64(b.LightMoonWeightEye),
	}
}
