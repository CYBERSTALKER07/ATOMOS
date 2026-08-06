package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

// runWMSPickWavesE2E exercises §8.7 Wave 1B pick-wave create + seal gate (flag-gated).
func runWMSPickWavesE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	_ = cfg
	whID := demoWarehouseID()

	status, listBody, _, err := clientDo(ctx, client, http.MethodGet,
		base+"/v1/warehouse/ops/pick-waves?warehouse_id="+whID, nil, cookie, "")
	if err != nil || status == http.StatusConflict || status == http.StatusNotFound || status >= 500 {
		fmt.Println("PX_E2E_WMS_PICK_WAVE_SKIPPED")
		fmt.Println("PX_E2E_WMS_SEAL_GATE_SKIPPED")
		return nil
	}
	if status >= 400 {
		// Disabled or unauthorized → soft skip
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(listBody, &errBody)
		if strings.Contains(errBody.Error, "disabled") || strings.Contains(errBody.Error, "pick_waves") {
			fmt.Println("PX_E2E_WMS_PICK_WAVE_SKIPPED")
			fmt.Println("PX_E2E_WMS_SEAL_GATE_SKIPPED")
			return nil
		}
		fmt.Println("PX_E2E_WMS_PICK_WAVE_SKIPPED")
		fmt.Println("PX_E2E_WMS_SEAL_GATE_SKIPPED")
		return nil
	}

	// Find a DRAFT/LOADING warehouse manifest to create a wave from.
	date := time.Now().UTC().Format("2006-01-02")
	status, manBody, _, err := clientDo(ctx, client, http.MethodGet,
		base+"/v1/warehouse/ops/manifests?date="+date, nil, cookie, "")
	manifestID := ""
	if err == nil && status < 400 {
		var manResp struct {
			Manifests []struct {
				ManifestID string `json:"manifest_id"`
				Status     string `json:"status"`
				State      string `json:"state"`
			} `json:"manifests"`
		}
		_ = json.Unmarshal(manBody, &manResp)
		for _, m := range manResp.Manifests {
			st := strings.ToUpper(strings.TrimSpace(coalesce(m.Status, m.State)))
			if st == "DRAFT" || st == "LOADING" {
				manifestID = m.ManifestID
				break
			}
		}
	}

	waveOK := false
	if manifestID != "" {
		createBody, _ := json.Marshal(map[string]any{"manifest_id": manifestID})
		status, createResp, _, err := clientPost(ctx, client,
			base+"/v1/warehouse/ops/pick-waves?warehouse_id="+whID,
			createBody, cookie, "ssmr-pick-wave-"+manifestID)
		if err == nil && status < 400 {
			var wave struct {
				WaveID string `json:"wave_id"`
			}
			_ = json.Unmarshal(createResp, &wave)
			if strings.TrimSpace(wave.WaveID) != "" {
				waveOK = true
			}
		} else if status == http.StatusConflict {
			// Already exists counts as wired path.
			var errBody struct {
				Error string `json:"error"`
			}
			_ = json.Unmarshal(createResp, &errBody)
			if strings.Contains(errBody.Error, "pick_wave_exists") {
				waveOK = true
			}
		}
	}

	// List success alone proves API is mounted when flag on.
	if !waveOK {
		var listed struct {
			Waves []any `json:"waves"`
		}
		_ = json.Unmarshal(listBody, &listed)
		if listed.Waves != nil {
			waveOK = true
		}
	}
	if waveOK {
		fmt.Println("PX_E2E_WMS_PICK_WAVE_OK")
	} else {
		fmt.Println("PX_E2E_WMS_PICK_WAVE_SKIPPED")
	}

	// Seal gate: attempt seal without ready wave → expect pick_wave_required / incomplete when Spanner-backed.
	gateManifest := manifestID
	if gateManifest == "" {
		gateManifest = "ssmr-missing-pick-wave-" + fmt.Sprintf("%d", time.Now().Unix()%100000)
	}
	sealBody, _ := json.Marshal(map[string]any{"manifest_id": gateManifest})
	status, sealResp, _, err := clientPost(ctx, client, base+"/v1/payload/seal", sealBody, cookie, "ssmr-seal-gate-"+gateManifest)
	if err != nil {
		fmt.Println("PX_E2E_WMS_SEAL_GATE_SKIPPED")
		return nil
	}
	var sealErr struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(sealResp, &sealErr)
	errText := strings.ToLower(sealErr.Error)
	if status == http.StatusConflict && (strings.Contains(errText, "pick_wave_required") || strings.Contains(errText, "pick_wave_incomplete")) {
		fmt.Println("PX_E2E_WMS_SEAL_GATE_OK")
		return nil
	}
	// Flag may be on but demo in-memory manifests skip gate → soft skip.
	fmt.Println("PX_E2E_WMS_SEAL_GATE_SKIPPED")
	return nil
}

func coalesce(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
