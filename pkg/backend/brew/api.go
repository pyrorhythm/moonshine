package brew

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pyrorhythm/moonshine/pkg/backend"
)

const apiBase = "https://formulae.brew.sh/api"

// errNotFound is returned when the API returns 404.
var errNotFound = errors.New("not found")

// apiClient fetches package metadata from formulae.brew.sh.
type apiClient struct {
	http *http.Client
}

func newAPIClient() *apiClient {
	return &apiClient{
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

// APIFormulaInfo is the response shape of /api/formula/{name}.json.
type APIFormulaInfo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Desc     string `json:"desc"`
	Tap      string `json:"tap"`
	Versions struct {
		Stable string `json:"stable"`
		Head   string `json:"head"`
		Bottle bool   `json:"bottle"`
	} `json:"versions"`
}

// APICaskInfo is the response shape of /api/cask/{token}.json.
type APICaskInfo struct {
	Token   string   `json:"token"`
	Name    []string `json:"name"`
	Desc    string   `json:"desc"`
	Version string   `json:"version"`
}

func (c *apiClient) get(ctx context.Context, rawURL string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("brew API %s: HTTP %d", rawURL, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

// FormulaInfo fetches metadata for a single formula by name.
func (c *apiClient) FormulaInfo(ctx context.Context, name string) (*APIFormulaInfo, error) {
	var info APIFormulaInfo
	err := c.get(ctx, apiBase+"/formula/"+url.PathEscape(name)+".json", &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// CaskInfo fetches metadata for a single cask by token.
func (c *apiClient) CaskInfo(ctx context.Context, token string) (*APICaskInfo, error) {
	var info APICaskInfo
	err := c.get(ctx, apiBase+"/cask/"+url.PathEscape(token)+".json", &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// PackageExists reports whether name is a known formula or cask.
func (c *apiClient) PackageExists(ctx context.Context, name string) (bool, error) {
	_, err := c.FormulaInfo(ctx, name)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, errNotFound) {
		return false, err
	}
	_, err = c.CaskInfo(ctx, name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, errNotFound) {
		return false, nil
	}
	return false, err
}

// Search fetches the full formula + cask lists and returns all entries whose
// name or token contains query (case-insensitive). Cask results are labelled
// with " [cask]" in the description so the caller can distinguish them.
func (c *apiClient) Search(ctx context.Context, query string) ([]backend.SearchResult, error) {
	q := strings.ToLower(query)

	var formulas []APIFormulaInfo
	if err := c.get(ctx, apiBase+"/formula.json", &formulas); err != nil {
		return nil, fmt.Errorf("fetching formula list: %w", err)
	}

	var casks []APICaskInfo
	// Non-fatal: cask search degrades gracefully.
	_ = c.get(ctx, apiBase+"/cask.json", &casks)

	var results []backend.SearchResult

	for _, f := range formulas {
		if strings.Contains(strings.ToLower(f.Name), q) {
			results = append(results, backend.SearchResult{
				Name:        f.Name,
				Version:     f.Versions.Stable,
				Description: f.Desc,
				Backend:     "brew",
			})
		}
	}
	for _, k := range casks {
		if strings.Contains(strings.ToLower(k.Token), q) {
			desc := k.Desc
			if desc != "" {
				desc += " [cask]"
			} else {
				desc = "[cask]"
			}
			results = append(results, backend.SearchResult{
				Name:        k.Token,
				Version:     k.Version,
				Description: desc,
				Backend:     "brew",
			})
		}
	}

	return results, nil
}
