package lockfile

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// LockFile records the exact state of every package moonshine has installed.
type LockFile struct {
	GeneratedAt time.Time                  `yaml:"generated_at"`
	Mode        string                     `yaml:"mode"`
	Packages    map[string][]LockedPackage `yaml:"packages"`
}

// LockedPackage is the record for one installed package.
type LockedPackage struct {
	Name        string    `yaml:"name"`
	Version     string    `yaml:"version"`
	Source      string    `yaml:"source"` // tap name, registry, etc.
	InstalledAt time.Time `yaml:"installed_at"`
}

// New returns an empty LockFile with the given mode.
func New(opMode string) *LockFile {
	return &LockFile{
		Mode:     opMode,
		Packages: make(map[string][]LockedPackage),
	}
}

// Load reads and parses a lockfile at path.
// If the file does not exist, an empty LockFile is returned.
func Load(path string) (*LockFile, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return New(""), nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading lockfile: %w", err)
	}
	var lf LockFile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parsing lockfile: %w", err)
	}
	if lf.Packages == nil {
		lf.Packages = make(map[string][]LockedPackage)
	}
	return &lf, nil
}

// Save writes the lockfile to path atomically.
func Save(path string, lf *LockFile) error {
	lf.GeneratedAt = time.Now().UTC()
	data, err := yaml.Marshal(lf)
	if err != nil {
		return fmt.Errorf("marshalling lockfile: %w", err)
	}
	tmp, err := os.CreateTemp("", "moonfile-lock-*.yaml")
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

// Upsert adds or updates a locked package entry for the given backend.
func (lf *LockFile) Upsert(backendName string, pkg LockedPackage) {
	pkgs := lf.Packages[backendName]
	for i, p := range pkgs {
		if p.Name == pkg.Name {
			pkgs[i] = pkg
			lf.Packages[backendName] = pkgs
			return
		}
	}
	lf.Packages[backendName] = append(pkgs, pkg)
}

// Remove deletes the lock entry for name under the given backend.
func (lf *LockFile) Remove(backendName, name string) {
	pkgs := lf.Packages[backendName]
	out := pkgs[:0]
	for _, p := range pkgs {
		if p.Name != name {
			out = append(out, p)
		}
	}
	lf.Packages[backendName] = out
}

// Contains reports whether name is recorded in the lockfile for the given backend.
func (lf *LockFile) Contains(backendName, name string) bool {
	for _, p := range lf.Packages[backendName] {
		if p.Name == name {
			return true
		}
	}
	return false
}

// Get returns the locked package for name under backend, if present.
func (lf *LockFile) Get(backendName, name string) (LockedPackage, bool) {
	for _, p := range lf.Packages[backendName] {
		if p.Name == name {
			return p, true
		}
	}
	return LockedPackage{}, false
}
