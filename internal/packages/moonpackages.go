package packages

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"
)

// moonpackagesYAML is the top-level structure of moonpackages.yml.
type moonpackagesYAML struct {
	Brew  []brewPkgYAML  `yaml:"brew,omitempty"`
	Go    []goPkgYAML    `yaml:"go,omitempty"`
	Cargo []cargoPkgYAML `yaml:"cargo,omitempty"`
	Npm   []npmPkgYAML   `yaml:"npm,omitempty"`
}

// brewPkgYAML represents one brew entry: a plain string or a struct.
//
//	- gcc               → name only
//	- name: openssl     → with optional version / brew_version / tap
//	  brew_version: 3
type brewPkgYAML struct {
	Name        string
	Version     string
	BrewVersion string
	Tap         string
}

func (b *brewPkgYAML) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		b.Name = value.Value
		return nil
	}
	type raw struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		BrewVersion string `yaml:"brew_version"`
		Tap         string `yaml:"tap"`
	}
	var r raw
	if err := value.Decode(&r); err != nil {
		return err
	}
	b.Name, b.Version, b.BrewVersion, b.Tap = r.Name, r.Version, r.BrewVersion, r.Tap
	return nil
}

func (b brewPkgYAML) MarshalYAML() (interface{}, error) {
	if b.Version == "" && b.BrewVersion == "" && b.Tap == "" {
		return b.Name, nil
	}
	type raw struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version,omitempty"`
		BrewVersion string `yaml:"brew_version,omitempty"`
		Tap         string `yaml:"tap,omitempty"`
	}
	return raw{b.Name, b.Version, b.BrewVersion, b.Tap}, nil
}

func (b brewPkgYAML) toPackage() Package {
	meta := map[string]string{"name": b.Name}
	if b.Version != "" {
		meta["version"] = b.Version
	}
	if b.BrewVersion != "" {
		meta["brew_version"] = b.BrewVersion
	}
	if b.Tap != "" {
		meta["tap"] = b.Tap
	}
	return Package{PackageManager: "brew", Meta: meta}
}

// goPkgYAML represents one go entry.
//
//	- golang.org/x/tools/gopls          → link only
//	- link: golang.org/x/tools/gopls    → with optional version
//	  version: latest
type goPkgYAML struct {
	Link    string
	Version string
}

func (g *goPkgYAML) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		g.Link = value.Value
		return nil
	}
	type raw struct {
		Link    string `yaml:"link"`
		Version string `yaml:"version"`
	}
	var r raw
	if err := value.Decode(&r); err != nil {
		return err
	}
	g.Link, g.Version = r.Link, r.Version
	return nil
}

func (g goPkgYAML) MarshalYAML() (interface{}, error) {
	if g.Version == "" {
		return g.Link, nil
	}
	type raw struct {
		Link    string `yaml:"link"`
		Version string `yaml:"version,omitempty"`
	}
	return raw{g.Link, g.Version}, nil
}

func (g goPkgYAML) toPackage() Package {
	meta := map[string]string{"link": g.Link}
	if g.Version != "" {
		meta["version"] = g.Version
	}
	return Package{PackageManager: "go", Meta: meta}
}

// cargoPkgYAML represents one cargo entry.
type cargoPkgYAML struct {
	Name    string
	Version string
}

func (c *cargoPkgYAML) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		c.Name = value.Value
		return nil
	}
	type raw struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	}
	var r raw
	if err := value.Decode(&r); err != nil {
		return err
	}
	c.Name, c.Version = r.Name, r.Version
	return nil
}

func (c cargoPkgYAML) MarshalYAML() (interface{}, error) {
	if c.Version == "" {
		return c.Name, nil
	}
	type raw struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version,omitempty"`
	}
	return raw{c.Name, c.Version}, nil
}

func (c cargoPkgYAML) toPackage() Package {
	meta := map[string]string{"name": c.Name}
	if c.Version != "" {
		meta["version"] = c.Version
	}
	return Package{PackageManager: "cargo", Meta: meta}
}

// npmPkgYAML represents one npm entry.
type npmPkgYAML struct {
	Name    string
	Version string
}

func (n *npmPkgYAML) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		n.Name = value.Value
		return nil
	}
	type raw struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	}
	var r raw
	if err := value.Decode(&r); err != nil {
		return err
	}
	n.Name, n.Version = r.Name, r.Version
	return nil
}

func (n npmPkgYAML) MarshalYAML() (interface{}, error) {
	if n.Version == "" {
		return n.Name, nil
	}
	type raw struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version,omitempty"`
	}
	return raw{n.Name, n.Version}, nil
}

func (n npmPkgYAML) toPackage() Package {
	meta := map[string]string{"name": n.Name}
	if n.Version != "" {
		meta["version"] = n.Version
	}
	return Package{PackageManager: "npm", Meta: meta}
}

// LoadMoonpackages reads moonpackages.yml. Returns empty list if the file does not exist.
func LoadMoonpackages(path string) (List, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return List{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading moonpackages.yml: %w", err)
	}
	var raw moonpackagesYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing moonpackages.yml: %w", err)
	}
	var list List
	for _, e := range raw.Brew {
		list = append(list, e.toPackage())
	}
	for _, e := range raw.Go {
		list = append(list, e.toPackage())
	}
	for _, e := range raw.Cargo {
		list = append(list, e.toPackage())
	}
	for _, e := range raw.Npm {
		list = append(list, e.toPackage())
	}
	return list, nil
}

// SaveMoonpackages writes list to path atomically.
func SaveMoonpackages(path string, list List) error {
	var raw moonpackagesYAML
	for _, pkg := range list {
		switch pkg.PackageManager {
		case "brew":
			raw.Brew = append(raw.Brew, brewPkgYAML{
				Name:        pkg.Get("name"),
				Version:     pkg.Get("version"),
				BrewVersion: pkg.Get("brew_version"),
				Tap:         pkg.Get("tap"),
			})
		case "go":
			raw.Go = append(raw.Go, goPkgYAML{
				Link:    pkg.Get("link"),
				Version: pkg.Get("version"),
			})
		case "cargo":
			raw.Cargo = append(raw.Cargo, cargoPkgYAML{
				Name:    pkg.Get("name"),
				Version: pkg.Get("version"),
			})
		case "npm":
			raw.Npm = append(raw.Npm, npmPkgYAML{
				Name:    pkg.Get("name"),
				Version: pkg.Get("version"),
			})
		}
	}

	data, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshalling moonpackages.yml: %w", err)
	}
	tmp, err := os.CreateTemp("", "moonpackages-*.yml")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	tmp.Close()
	return os.Rename(tmp.Name(), path)
}
