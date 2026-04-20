package packages

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
)

// canonicalKeyOrder defines the preferred key output order per backend.
var canonicalKeyOrder = map[string][]string{
	"brew":  {"name", "version", "tap"},
	"go":    {"module", "path", "version"},
	"cargo": {"name", "version"},
	"npm":   {"name", "version"},
}

// Parse parses moonpackages content.
// Each non-blank, non-comment line must start with "--" followed by key=value pairs.
func Parse(data []byte) (List, error) {
	var list List
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "--") {
			return nil, fmt.Errorf("line %d: expected entry starting with '--'", lineNum)
		}
		rest := strings.TrimSpace(line[2:])
		meta, err := parseKV(rest)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		pm := meta["package_manager"]
		if pm == "" {
			return nil, fmt.Errorf("line %d: missing package_manager", lineNum)
		}
		delete(meta, "package_manager")
		list = append(list, Package{PackageManager: pm, Meta: meta})
	}
	return list, scanner.Err()
}

// Format serializes a List to moonpackages text, grouped by package_manager.
func Format(list List) []byte {
	type group struct {
		name string
		pkgs []Package
	}
	var groups []group
	index := make(map[string]int)
	for _, pkg := range list {
		i, ok := index[pkg.PackageManager]
		if !ok {
			i = len(groups)
			groups = append(groups, group{name: pkg.PackageManager})
			index[pkg.PackageManager] = i
		}
		groups[i].pkgs = append(groups[i].pkgs, pkg)
	}

	var buf bytes.Buffer
	for gi, g := range groups {
		if gi > 0 {
			buf.WriteByte('\n')
		}
		for _, pkg := range g.pkgs {
			buf.WriteString("-- package_manager=")
			buf.WriteString(g.name)
			for _, k := range orderedKeys(g.name, pkg.Meta) {
				v := pkg.Meta[k]
				if v == "" {
					continue
				}
				buf.WriteByte(' ')
				buf.WriteString(k)
				buf.WriteByte('=')
				if strings.ContainsAny(v, " \t") {
					buf.WriteByte('"')
					buf.WriteString(v)
					buf.WriteByte('"')
				} else {
					buf.WriteString(v)
				}
			}
			buf.WriteByte('\n')
		}
	}
	return buf.Bytes()
}

// LoadMoonpackages reads a moonpackages file. Returns empty list if the file does not exist.
func LoadMoonpackages(path string) (List, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return List{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading moonpackages: %w", err)
	}
	list, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parsing moonpackages: %w", err)
	}
	return list, nil
}

// SaveMoonpackages writes list to path atomically.
func SaveMoonpackages(path string, list List) error {
	data := Format(list)
	tmp, err := os.CreateTemp("", "moonpackages-*")
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

// parseKV parses a string of space-separated key=value pairs.
// Values may be double-quoted to include spaces.
func parseKV(s string) (map[string]string, error) {
	meta := make(map[string]string)
	for {
		s = strings.TrimLeft(s, " \t")
		if s == "" {
			break
		}
		eq := strings.IndexByte(s, '=')
		if eq < 0 {
			return nil, fmt.Errorf("expected key=value, got %q", s)
		}
		key := strings.TrimSpace(s[:eq])
		s = s[eq+1:]
		var value string
		if len(s) > 0 && s[0] == '"' {
			end := strings.IndexByte(s[1:], '"')
			if end < 0 {
				return nil, fmt.Errorf("unterminated quote for key %q", key)
			}
			value = s[1 : end+1]
			s = s[end+2:]
		} else {
			sp := strings.IndexAny(s, " \t")
			if sp < 0 {
				value = s
				s = ""
			} else {
				value = s[:sp]
				s = s[sp:]
			}
		}
		meta[key] = value
	}
	return meta, nil
}

// orderedKeys returns meta keys in canonical output order for the given backend.
func orderedKeys(pm string, meta map[string]string) []string {
	order := canonicalKeyOrder[pm]
	seen := make(map[string]bool, len(meta))
	var keys []string
	for _, k := range order {
		if _, ok := meta[k]; ok {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	var extra []string
	for k := range meta {
		if !seen[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	return append(keys, extra...)
}
