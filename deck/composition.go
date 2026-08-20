package deck

const CompositionCatalogVersion = "r1"

type CompositionName string

type CompositionSlot struct {
	Name       string
	Role       string
	Required   bool
	SourceLine int
}

type CompositionSpec struct {
	CatalogVersion string
	Name           CompositionName
	LayoutClass    string
	Variant        string
	MinSlots       int
	MaxSlots       int
	Slots          []CompositionSlot
	BodyRole       string
}

var compositionCatalog = map[CompositionName]CompositionSpec{
	"content": {
		CatalogVersion: CompositionCatalogVersion,
		Name:           "content",
		Variant:        "content",
		BodyRole:       "body",
	},
	"agenda": {
		CatalogVersion: CompositionCatalogVersion,
		Name:           "agenda",
		LayoutClass:    "timeline",
		Variant:        "agenda",
		MinSlots:       3,
		MaxSlots:       6,
		Slots:          numberedCompositionSlots("item-", "agenda", 6, 3),
	},
	"media-split": {
		CatalogVersion: CompositionCatalogVersion,
		Name:           "media-split",
		LayoutClass:    "columns",
		Variant:        "split",
		MinSlots:       2,
		MaxSlots:       2,
		Slots: []CompositionSlot{
			{Name: "media", Role: "media", Required: true},
			{Name: "content", Role: "content", Required: true},
		},
	},
	"media-stage": {
		CatalogVersion: CompositionCatalogVersion,
		Name:           "media-stage",
		LayoutClass:    "columns",
		Variant:        "stage",
		MinSlots:       2,
		MaxSlots:       2,
		Slots: []CompositionSlot{
			{Name: "media", Role: "media", Required: true},
			{Name: "content", Role: "content", Required: true},
		},
	},
	"steps": {
		CatalogVersion: CompositionCatalogVersion,
		Name:           "steps",
		LayoutClass:    "timeline",
		Variant:        "steps",
		MinSlots:       3,
		MaxSlots:       6,
		Slots:          numberedCompositionSlots("step-", "step", 6, 3),
	},
	"highlight": {
		CatalogVersion: CompositionCatalogVersion,
		Name:           "highlight",
		LayoutClass:    "section",
		Variant:        "highlight",
		BodyRole:       "body",
	},
	"compare-grid": {
		CatalogVersion: CompositionCatalogVersion,
		Name:           "compare-grid",
		LayoutClass:    "grid",
		Variant:        "compare",
		MinSlots:       2,
		MaxSlots:       4,
		Slots:          numberedCompositionSlots("item-", "item", 4, 2),
	},
	"hero": {
		CatalogVersion: CompositionCatalogVersion,
		Name:           "hero",
		LayoutClass:    "lead",
		Variant:        "hero",
		BodyRole:       "body",
	},
	"image-grid": {
		CatalogVersion: CompositionCatalogVersion,
		Name:           "image-grid",
		LayoutClass:    "grid",
		Variant:        "image",
		MinSlots:       2,
		MaxSlots:       4,
		Slots:          numberedCompositionSlots("image-", "media", 4, 2),
	},
}

func isCompositionName(value string) bool {
	if value == "none" {
		return true
	}
	_, ok := compositionCatalog[CompositionName(value)]
	return ok
}

func numberedCompositionSlots(prefix, role string, max, required int) []CompositionSlot {
	slots := make([]CompositionSlot, max)
	for index := range slots {
		slots[index] = CompositionSlot{
			Name:     prefix + string(rune('1'+index)),
			Role:     role,
			Required: index < required,
		}
	}
	return slots
}

func ResolveComposition(name CompositionName) (CompositionSpec, error) {
	if name == "" || name == "none" {
		if name == "" {
			return CompositionSpec{}, deckError("deck.composition_invalid", "", 1, "composition name is empty")
		}
		return CompositionSpec{}, nil
	}
	spec, ok := compositionCatalog[name]
	if !ok {
		return CompositionSpec{}, deckError("deck.composition_invalid", "", 1, "unsupported composition "+string(name))
	}
	return cloneCompositionSpec(spec), nil
}

func cloneCompositionSpec(spec CompositionSpec) CompositionSpec {
	spec.Slots = append([]CompositionSlot(nil), spec.Slots...)
	return spec
}
