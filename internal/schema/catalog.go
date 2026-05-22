package schema

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const DefaultCatalogURL = "https://www.schemastore.org/api/json/catalog.json"

type catalogEntry struct {
	Name      string   `json:"name"`
	URL       string   `json:"url"`
	FileMatch []string `json:"fileMatch"`
}

type catalogDoc struct {
	Schemas []catalogEntry `json:"schemas"`
}

type Catalog struct {
	URL    string
	Client *http.Client

	once    sync.Once
	loadErr error
	entries []catalogEntry
}

func NewCatalog(url string) *Catalog {
	if url == "" {
		url = DefaultCatalogURL
	}
	return &Catalog{
		URL:    url,
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Catalog) Match(docPath string) string {
	if docPath == "" {
		return ""
	}
	c.once.Do(c.load)
	if c.loadErr != nil {
		return ""
	}
	for _, e := range c.entries {
		for _, pat := range e.FileMatch {
			if matchGlob(pat, docPath) {
				return e.URL
			}
			// Catalog patterns commonly omit a leading `**/`.
			if !startsWithStar(pat) && matchGlob("**/"+pat, docPath) {
				return e.URL
			}
		}
	}
	return ""
}

func startsWithStar(pat string) bool {
	return len(pat) > 0 && pat[0] == '*'
}

func (c *Catalog) load() {
	req, err := http.NewRequest(http.MethodGet, c.URL, nil)
	if err != nil {
		c.loadErr = err
		return
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		c.loadErr = err
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		c.loadErr = fmt.Errorf("catalog: HTTP %d", resp.StatusCode)
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.loadErr = err
		return
	}
	var doc catalogDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		c.loadErr = err
		return
	}
	c.entries = doc.Schemas
}
