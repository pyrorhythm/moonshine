package brew


// FormulaInfo is parsed from `brew info --json=v2`.
type FormulaInfo struct {
	Name      string             `json:"name"`
	FullName  string             `json:"full_name"`
	Tap       string             `json:"tap"`
	Versions  FormulaVersions    `json:"versions"`
	Installed []InstalledVersion `json:"installed"`
}

// FormulaVersions holds stable/head version strings.
type FormulaVersions struct {
	Stable string `json:"stable"`
	Head   string `json:"head"`
	Bottle bool   `json:"bottle"`
}

// InstalledVersion is one installed version entry inside FormulaInfo.
type InstalledVersion struct {
	Version       string   `json:"version"`
	UsedOptions   []string `json:"used_options"`
	BuiltAsBottle bool     `json:"built_as_bottle"`
}

// infoV2Response is the top-level structure returned by `brew info --json=v2`.
type infoV2Response struct {
	Formulae []FormulaInfo `json:"formulae"`
}