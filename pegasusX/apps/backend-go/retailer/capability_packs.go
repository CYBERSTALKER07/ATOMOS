package retailer

import (
	"fmt"
	"sort"
	"strings"
)

// Capability pack identifiers (Retail OS).
const (
	PackCORE           = "CORE"
	PackTEAM           = "TEAM"
	PackLOCATIONS      = "LOCATIONS"
	PackSTORESTOCK     = "STORE_STOCK"
	PackSECTIONS       = "SECTIONS"
	PackPOS            = "POS"
	PackSHIFTS         = "SHIFTS"
	PackREPORTSPRO     = "REPORTS_PRO"
	PackCUSTOMERASSIST = "CUSTOMER_ASSIST"
)

// PackMeta describes a capability pack for API/UI.
type PackMeta struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	HardDeps    []string `json:"hard_deps"`
	SoftDeps    []string `json:"soft_deps"`
	AlwaysOn    bool     `json:"always_on"`
}

// PackCatalog is the ordered catalog of packs.
var PackCatalog = []PackMeta{
	{ID: PackCORE, Name: "Core procurement", Description: "Catalog, cart, checkout, orders, dock, claims, credit, notifications", AlwaysOn: true},
	{ID: PackTEAM, Name: "Team & access", Description: "Staff invites, roles, permission-aware navigation", SoftDeps: []string{}},
	{ID: PackLOCATIONS, Name: "Multi-location", Description: "Branches, location switcher, per-location receiving windows", SoftDeps: []string{PackTEAM}},
	{ID: PackSTORESTOCK, Name: "Store inventory", Description: "Backroom/floor stock ledger, receive, counts, transfers", SoftDeps: []string{PackTEAM}},
	{ID: PackSECTIONS, Name: "Sections & shelves", Description: "Departments, shelf labels, section staff", HardDeps: []string{PackSTORESTOCK}, SoftDeps: []string{PackTEAM}},
	{ID: PackPOS, Name: "Point of sale", Description: "Registers, sales, tenders, voids", HardDeps: []string{PackSTORESTOCK}, SoftDeps: []string{PackTEAM, PackSHIFTS}},
	{ID: PackSHIFTS, Name: "Shifts & time", Description: "Clock in/out, open/close shift, cash recon", HardDeps: []string{PackTEAM}, SoftDeps: []string{PackPOS}},
	{ID: PackREPORTSPRO, Name: "Ops reports", Description: "Sales, inventory, shrinkage, multi-location rollup", SoftDeps: []string{}},
	{ID: PackCUSTOMERASSIST, Name: "Floor assist", Description: "Section help queue for large stores", HardDeps: []string{PackSECTIONS, PackTEAM}},
}

var packByID map[string]PackMeta

func init() {
	packByID = make(map[string]PackMeta, len(PackCatalog))
	for _, p := range PackCatalog {
		packByID[p.ID] = p
	}
}

// NormalizePackID uppercases and trims a pack id.
func NormalizePackID(id string) string {
	return strings.ToUpper(strings.TrimSpace(id))
}

// KnownPack reports whether pack id is in the catalog.
func KnownPack(id string) bool {
	_, ok := packByID[NormalizePackID(id)]
	return ok
}

// EnabledSet is a set of enabled pack ids (CORE always implied).
type EnabledSet map[string]bool

// WithCORE ensures CORE is present.
func (s EnabledSet) WithCORE() EnabledSet {
	if s == nil {
		s = EnabledSet{}
	}
	s[PackCORE] = true
	return s
}

// List returns sorted enabled pack ids including CORE.
func (s EnabledSet) List() []string {
	s = s.WithCORE()
	out := make([]string, 0, len(s))
	for id, on := range s {
		if on {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// Has reports whether pack is enabled (CORE always true).
func (s EnabledSet) Has(id string) bool {
	id = NormalizePackID(id)
	if id == PackCORE {
		return true
	}
	return s != nil && s[id]
}

// EnableResult is the outcome of attempting to enable a pack.
type EnableResult struct {
	Status       string   `json:"status"` // OK | BLOCKED | WARN
	PackID       string   `json:"pack_id"`
	MissingHard  []string `json:"missing_hard_deps,omitempty"`
	MissingSoft  []string `json:"missing_soft_deps,omitempty"`
	EnableAll    []string `json:"enable_all,omitempty"`
	Message      string   `json:"message,omitempty"`
	WouldEnable  []string `json:"would_enable,omitempty"`
}

// EvaluateEnable checks dependency graph for enabling packID.
// acceptSoft: if false and soft deps missing → WARN; if true → OK and soft included in WouldEnable only when forceSoft.
func EvaluateEnable(enabled EnabledSet, packID string, acceptSoft bool) EnableResult {
	packID = NormalizePackID(packID)
	meta, ok := packByID[packID]
	if !ok {
		return EnableResult{Status: "BLOCKED", PackID: packID, Message: "unknown_pack"}
	}
	if meta.AlwaysOn || packID == PackCORE {
		return EnableResult{Status: "OK", PackID: packID, Message: "always_on", WouldEnable: []string{PackCORE}}
	}
	enabled = enabled.WithCORE()
	if enabled.Has(packID) {
		return EnableResult{Status: "OK", PackID: packID, Message: "already_enabled", WouldEnable: []string{packID}}
	}

	var missingHard, missingSoft []string
	for _, d := range meta.HardDeps {
		if !enabled.Has(d) {
			missingHard = append(missingHard, d)
		}
	}
	for _, d := range meta.SoftDeps {
		if !enabled.Has(d) {
			missingSoft = append(missingSoft, d)
		}
	}
	// hard deps of hard deps (one level is enough for current graph; expand transitively)
	missingHard = expandHardMissing(enabled, missingHard)

	if len(missingHard) > 0 {
		all := append([]string{}, missingHard...)
		all = append(all, packID)
		return EnableResult{
			Status:      "BLOCKED",
			PackID:      packID,
			MissingHard: missingHard,
			MissingSoft: missingSoft,
			EnableAll:   uniqueSorted(all),
			Message:     fmt.Sprintf("%s requires %s", packID, strings.Join(missingHard, ", ")),
			WouldEnable: uniqueSorted(all),
		}
	}
	if len(missingSoft) > 0 && !acceptSoft {
		all := append([]string{}, missingSoft...)
		all = append(all, packID)
		return EnableResult{
			Status:      "WARN",
			PackID:      packID,
			MissingSoft: missingSoft,
			EnableAll:   uniqueSorted(all),
			Message:     fmt.Sprintf("%s works better with %s", packID, strings.Join(missingSoft, ", ")),
			WouldEnable: []string{packID},
		}
	}
	return EnableResult{Status: "OK", PackID: packID, WouldEnable: []string{packID}}
}

func expandHardMissing(enabled EnabledSet, missing []string) []string {
	seen := map[string]bool{}
	var out []string
	queue := append([]string{}, missing...)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		id = NormalizePackID(id)
		if seen[id] || enabled.Has(id) {
			continue
		}
		seen[id] = true
		out = append(out, id)
		if meta, ok := packByID[id]; ok {
			for _, d := range meta.HardDeps {
				if !enabled.Has(d) && !seen[d] {
					queue = append(queue, d)
				}
			}
		}
	}
	return uniqueSorted(out)
}

// EvaluateDisable checks whether pack can be disabled.
func EvaluateDisable(enabled EnabledSet, packID string) EnableResult {
	packID = NormalizePackID(packID)
	meta, ok := packByID[packID]
	if !ok {
		return EnableResult{Status: "BLOCKED", PackID: packID, Message: "unknown_pack"}
	}
	if meta.AlwaysOn || packID == PackCORE {
		return EnableResult{Status: "BLOCKED", PackID: packID, Message: "cannot_disable_core"}
	}
	enabled = enabled.WithCORE()
	if !enabled.Has(packID) {
		return EnableResult{Status: "OK", PackID: packID, Message: "already_disabled"}
	}
	var dependents []string
	for _, p := range PackCatalog {
		if p.ID == packID {
			continue
		}
		if !enabled.Has(p.ID) {
			continue
		}
		for _, d := range p.HardDeps {
			if NormalizePackID(d) == packID {
				dependents = append(dependents, p.ID)
			}
		}
	}
	if len(dependents) > 0 {
		return EnableResult{
			Status:      "BLOCKED",
			PackID:      packID,
			Message:     fmt.Sprintf("disable %s first", strings.Join(dependents, ", ")),
			EnableAll:   uniqueSorted(dependents),
			WouldEnable: uniqueSorted(dependents),
		}
	}
	return EnableResult{Status: "OK", PackID: packID, WouldEnable: []string{packID}}
}

func uniqueSorted(in []string) []string {
	m := map[string]bool{}
	var out []string
	for _, s := range in {
		s = NormalizePackID(s)
		if s == "" || m[s] {
			continue
		}
		m[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
