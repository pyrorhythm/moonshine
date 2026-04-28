package packages

import (
	"fmt"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

type packagesYAML struct {
	Brew  []brewPkgYAML  `yaml:"brew,omitempty"`
	Go    []goPkgYAML    `yaml:"go,omitempty"`
	Cargo []cargoPkgYAML `yaml:"cargo,omitempty"`
	Npm   []npmPkgYAML   `yaml:"npm,omitempty"`
}

// brewPkgYAML represents one brew entry: a plain string or a struct.
//
//nolint:recvcheck
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

func (b brewPkgYAML) MarshalYAML() (any, error) {
	if b.Version == "" && b.BrewVersion == "" && b.Tap == "" {
		return b.Name, nil
	}
	type raw struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version,omitempty"`
		BrewVersion string `yaml:"brew_version,omitempty"`
		Tap         string `yaml:"tap,omitempty"`
	}
	return raw(b), nil
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

// goPkgYAML represents one go entry: a plain link string or a struct.
//
//nolint:recvcheck
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

func (g goPkgYAML) MarshalYAML() (any, error) {
	if g.Version == "" {
		return g.Link, nil
	}
	type raw struct {
		Link    string `yaml:"link"`
		Version string `yaml:"version,omitempty"`
	}
	return raw(g), nil
}

func (g goPkgYAML) toPackage() Package {
	meta := map[string]string{"link": g.Link}
	if g.Version != "" {
		meta["version"] = g.Version
	}
	return Package{PackageManager: "go", Meta: meta}
}

// cargoPkgYAML represents one cargo entry.
//
//nolint:recvcheck
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

func (c cargoPkgYAML) MarshalYAML() (any, error) {
	if c.Version == "" {
		return c.Name, nil
	}
	type raw struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version,omitempty"`
	}
	return raw(c), nil
}

func (c cargoPkgYAML) toPackage() Package {
	meta := map[string]string{"name": c.Name}
	if c.Version != "" {
		meta["version"] = c.Version
	}
	return Package{PackageManager: "cargo", Meta: meta}
}

// npmPkgYAML represents one npm entry.
//
//nolint:recvcheck
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

func (n npmPkgYAML) MarshalYAML() (any, error) {
	if n.Version == "" {
		return n.Name, nil
	}
	type raw struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version,omitempty"`
	}
	return raw(n), nil
}

func (n npmPkgYAML) toPackage() Package {
	meta := map[string]string{"name": n.Name}
	if n.Version != "" {
		meta["version"] = n.Version
	}
	return Package{PackageManager: "npm", Meta: meta}
}

// LoadPackages reads packages.yml. Returns empty list if the file does not exist.
func LoadPackages(path string) (List, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return List{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading packages file: %w", err)
	}
	var raw packagesYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing packages file: %w", err)
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

// SavePackages writes list to path atomically.
func SavePackages(path string, list List) error {
	var raw packagesYAML
	for _, pkg := range list {
		switch pkg.PackageManager {
		case "brew":
			name := pkg.Get("name")
			brewVer := pkg.Get("brew_version")
			if brewVer == "" {
				if idx := strings.LastIndexByte(name, '@'); idx >= 0 {
					brewVer = name[idx+1:]
					name = name[:idx]
				}
			}
			raw.Brew = append(raw.Brew, brewPkgYAML{
				Name:        name,
				Version:     pkg.Get("version"),
				BrewVersion: brewVer,
				Tap:         pkg.Get("tap"),
			})
		case "go":
			link := pkg.Get("link")
			if link == "" {
				continue
			}
			raw.Go = append(raw.Go, goPkgYAML{Link: link, Version: pkg.Get("version")})
		case "cargo":
			raw.Cargo = append(raw.Cargo, cargoPkgYAML{Name: pkg.Get("name"), Version: pkg.Get("version")})
		case "npm":
			raw.Npm = append(raw.Npm, npmPkgYAML{Name: pkg.Get("name"), Version: pkg.Get("version")})
		}
	}

	data, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshalling packages file: %w", err)
	}
	tmp, err := os.CreateTemp("", "packages-*.yml")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	_ = tmp.Close()
	return os.Rename(tmp.Name(), path)
}
