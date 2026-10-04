package items

// ToolType names the job a tool does. A gathering or processing job asks for
// one ToolType and is served by the best tool of that type the character has.
type ToolType string

const (
	ToolKnife        ToolType = "knife"         // skinning, cutting meat
	ToolCleaver      ToolType = "cleaver"       // butchering through joints: bone, fat
	ToolBoneSaw      ToolType = "bone_saw"      // horn, antler, tusk
	ToolAxe          ToolType = "axe"           // felling trees
	ToolSaw          ToolType = "saw"           // logs into planks, staves and shafts
	ToolScraper      ToolType = "scraper"       // fleshing hides before curing
	ToolSickle       ToolType = "sickle"        // herbs and fibre
	ToolCarvingKnife ToolType = "carving_knife" // bone, horn and fine woodwork
	ToolTrowel       ToolType = "trowel"        // roots and tubers
)

// AllToolTypes is every ToolType, in a stable order (validation, help text).
var AllToolTypes = []ToolType{
	ToolKnife, ToolCleaver, ToolBoneSaw, ToolAxe, ToolSaw,
	ToolScraper, ToolSickle, ToolCarvingKnife, ToolTrowel,
}

// ToolTier is how good a tool is. It multiplies the user's stat term on a
// gathering roll (Balance.ToolMult*) and caps the best material grade the
// tool can produce (MaxGrade).
type ToolTier int

const (
	ToolTierNone       ToolTier = 0
	ToolTierCrude      ToolTier = 1 // improvised: a sword used as a knife, a stone axe
	ToolTierIron       ToolTier = 2
	ToolTierSteel      ToolTier = 3
	ToolTierMasterwork ToolTier = 4
)

var toolTierNames = map[ToolTier]string{
	ToolTierCrude:      `crude`,
	ToolTierIron:       `iron`,
	ToolTierSteel:      `steel`,
	ToolTierMasterwork: `masterwork`,
}

func (t ToolTier) String() string { return toolTierNames[t] }

// Valid reports whether t is crude through masterwork.
func (t ToolTier) Valid() bool { return t >= ToolTierCrude && t <= ToolTierMasterwork }

// MaxGrade is the best material grade a tool of this tier can produce. A crude
// tool tops out at standard; only a masterwork tool can reach pristine.
func (t ToolTier) MaxGrade() Quality {
	switch t {
	case ToolTierCrude:
		return QualityStandard
	case ToolTierIron:
		return QualityFine
	case ToolTierSteel:
		return QualitySuperb
	case ToolTierMasterwork:
		return QualityPristine
	}
	return QualityStandard
}

// ToolSpec marks an item as a tool. It is authored on the item spec:
//
//	tool:
//	  type: knife
//	  tier: 3        # 1 crude, 2 iron, 3 steel, 4 masterwork
//	  speed: 1.25    # optional; >1 finishes jobs faster (default 1.0)
//
// An item with no ToolSpec may still serve as an IMPROVISED tool: see
// ImprovisedTool.
type ToolSpec struct {
	Type  ToolType `yaml:"type"`
	Tier  ToolTier `yaml:"tier"`
	Speed float64  `yaml:"speed,omitempty"`
}

// IsKnownToolType reports whether t is one of AllToolTypes.
func IsKnownToolType(t ToolType) bool {
	for _, k := range AllToolTypes {
		if k == t {
			return true
		}
	}
	return false
}

// EffectiveToolTier is the tier of THIS tool instance: the authored tier,
// nudged by the instance's own grade when it was crafted. A pristine iron
// knife works like steel; a crude steel one works like iron. Ungraded tools
// use the authored tier unchanged.
func EffectiveToolTier(authored ToolTier, grade Quality) ToolTier {
	t := authored
	switch grade {
	case QualityPristine:
		t++
	case QualityCrude:
		t--
	}
	if t < ToolTierCrude {
		t = ToolTierCrude
	}
	if t > ToolTierMasterwork {
		t = ToolTierMasterwork
	}
	return t
}

// ImprovisedTool reports whether a spec WITHOUT a ToolSpec can stand in for a
// tool of type want, and at what tier (always crude). Only one-handed weapons
// qualify, and only for the jobs their edge suits:
//
//   - knife: a stabbing or slashing weapon (dagger, short sword)
//   - cleaver and axe: a cleaving weapon (hatchet, war axe)
//
// Two-handed weapons never qualify: nobody skins a deer with a greatsword.
func ImprovisedTool(spec ItemSpec, want ToolType) (ToolTier, bool) {
	if spec.Tool != nil || spec.Type != Weapon || spec.Hands == TwoHanded {
		return ToolTierNone, false
	}
	switch want {
	case ToolKnife:
		if spec.Subtype == Stabbing || spec.Subtype == Slashing {
			return ToolTierCrude, true
		}
	case ToolCleaver, ToolAxe:
		if spec.Subtype == Cleaving {
			return ToolTierCrude, true
		}
	}
	return ToolTierNone, false
}
