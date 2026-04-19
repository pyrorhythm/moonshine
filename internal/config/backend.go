package config

// ShellBackendConfig defines a user-supplied registry driven by shell commands.
// (@pyrorhythm) ideally must be an outer package member
type ShellBackendConfig struct {
	Name          string `yaml:"name"`
	List          string `yaml:"list"`
	Install       string `yaml:"install"`
	InstallLatest string `yaml:"install_latest"`
	Uninstall     string `yaml:"uninstall"`
	Upgrade       string `yaml:"upgrade"`
}
