package auth

import (
	"os"
	"strings"
)

// Cell directory statuses. planned ≠ a live GCP cell.
const (
	CellStatusShipped = "shipped" // UZ/ssmr is the only live cell
	CellStatusPlanned = "planned"
)

// CellDirectoryEntry is the public API hostname for a home_cell (GS-C5).
// No secrets. planned rows are DNS names on paper, not a second live cell.
type CellDirectoryEntry struct {
	ID          string `json:"id"`
	APIHostname string `json:"api_hostname"`
	APIURL      string `json:"api_url"`
	WSURL       string `json:"ws_url"`
	Status      string `json:"status"`
	Live        bool   `json:"live"`
}

func catalogCell(id, host, status string, live bool) CellDirectoryEntry {
	api := "https://" + host
	return CellDirectoryEntry{
		ID:          id,
		APIHostname: host,
		APIURL:      api,
		WSURL:       wsURLFromAPI(api),
		Status:      status,
		Live:        live,
	}
}

// ListCells is the in-repo cell → API URL map. Matches terraform api_hostname
// (cells/uz, cells/eu) and global DNS records. EU/US/KZ are not applied.
func ListCells() []CellDirectoryEntry {
	return []CellDirectoryEntry{
		applyPublicBaseOverride(catalogCell("cell-uz", "api.pegasusx.app", CellStatusShipped, true)),
		applyPublicBaseOverride(catalogCell("cell-eu", "api-eu.pegasusx.app", CellStatusPlanned, false)),
		catalogCell("cell-us", "api-us.pegasusx.app", CellStatusPlanned, false),
		catalogCell("cell-kz", "api-kz.pegasusx.app", CellStatusPlanned, false),
	}
}

// ResolveCell returns the directory row for a home_cell id.
func ResolveCell(id string) (CellDirectoryEntry, bool) {
	want := strings.ToLower(strings.TrimSpace(id))
	if want == "" {
		return CellDirectoryEntry{}, false
	}
	for _, c := range ListCells() {
		if c.ID == want {
			return c, true
		}
	}
	return CellDirectoryEntry{}, false
}

// APIURLForHomeCell is the URL clients must call for that cell.
// PUBLIC_BASE_URL overrides the process HOME_CELL only (local/ssmr stay local).
func APIURLForHomeCell(id string) string {
	c, ok := ResolveCell(id)
	if !ok {
		return ""
	}
	return c.APIURL
}

// WSURLForHomeCell is the WebSocket origin for that cell.
func WSURLForHomeCell(id string) string {
	c, ok := ResolveCell(id)
	if !ok {
		return ""
	}
	return c.WSURL
}

func applyPublicBaseOverride(c CellDirectoryEntry) CellDirectoryEntry {
	home := DefaultHomeCellFromEnv()
	if c.ID != home {
		return c
	}
	pub := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")
	if pub == "" {
		return c
	}
	c.APIURL = pub
	c.WSURL = wsURLFromAPI(pub)
	return c
}

func wsURLFromAPI(api string) string {
	switch {
	case strings.HasPrefix(api, "https://"):
		return "wss://" + strings.TrimPrefix(api, "https://")
	case strings.HasPrefix(api, "http://"):
		return "ws://" + strings.TrimPrefix(api, "http://")
	default:
		return api
	}
}
