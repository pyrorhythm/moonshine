package mode

// OperatingMode
type OperatingMode string

const (
	Standalone OperatingMode = "standalone"
	Companion  OperatingMode = "companion"

	Default = Standalone
)

func (m OperatingMode) Valid() bool {
	return m == Standalone || m == Companion
}
