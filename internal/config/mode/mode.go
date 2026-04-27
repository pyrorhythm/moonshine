package mode

// OperatingMode is the mode in which moonshine manages packages.
type OperatingMode string

const (
	Standalone OperatingMode = "standalone"
	Companion  OperatingMode = "companion"

	Default = Standalone
)

func (m OperatingMode) Valid() bool {
	return m == Standalone || m == Companion
}
