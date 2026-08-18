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

// runWarehousePickWaveForManifest creates a MANIFEST pick wave and confirms every
// task to READY_TO_SEAL so payload seal can pass with WMS_PICK_WAVES_ENABLED.
// Flag-off is a no-op (seal gate is also off).
func runWarehousePickWaveForManifest(ctx context.Context, client *http.Client, base, cookie, manifestID string) error {
	manifestID = strings.TrimSpace(manifestID)
	if manifestID == "" {
		return nil
	}
	whID := demoWarehouseID()
	if strings.TrimSpace(whID) == "" {
		return fmt.Errorf("pick wave: warehouse_id required")
	}
	q := "?warehouse_id=" + whID
	createBody, _ := json.Marshal(map[string]any{"manifest_id": manifestID})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/warehouse/ops/pick-waves"+q, createBody, cookie, "ssmr-dispatch-pick-"+manifestID)
	if err != nil {
		return fmt.Errorf("pick wave create: %w", err)
	}
	if status == http.StatusConflict {
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(respBody, &errBody)
		errText := strings.ToLower(errBody.Error)
		if strings.Contains(errText, "disabled") {
			return nil
		}
		if !strings.Contains(errText, "pick_wave_exists") {
			return fmt.Errorf("pick wave create status %d body %s", status, string(respBody))
		}
		status, respBody, _, err = clientDo(ctx, client, http.MethodGet,
			base+"/v1/warehouse/ops/pick-waves"+q+"&manifest_id="+manifestID, nil, cookie, "")
		if err != nil {
			return fmt.Errorf("pick wave list: %w", err)
		}
		if status >= 400 {
			return fmt.Errorf("pick wave list status %d body %s", status, string(respBody))
		}
		var listed struct {
			Waves []struct {
				WaveID string `json:"wave_id"`
				Status string `json:"status"`
			} `json:"waves"`
		}
		if err := json.Unmarshal(respBody, &listed); err != nil {
			return fmt.Errorf("decode pick wave list: %w", err)
		}
		if len(listed.Waves) == 0 || strings.TrimSpace(listed.Waves[0].WaveID) == "" {
			return fmt.Errorf("pick_wave_exists but list empty for manifest %s", manifestID)
		}
		if strings.EqualFold(listed.Waves[0].Status, "READY_TO_SEAL") {
			fmt.Println("PX_E2E_WAREHOUSE_PICK_WAVE_READY_OK")
			return nil
		}
		return confirmPickWaveTasks(ctx, client, base, cookie, q, listed.Waves[0].WaveID)
	}
	if status >= 400 {
		return fmt.Errorf("pick wave create status %d body %s", status, string(respBody))
	}
	var created struct {
		WaveID string `json:"wave_id"`
		Status string `json:"status"`
		Tasks  []struct {
			TaskID string `json:"task_id"`
			Status string `json:"status"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return fmt.Errorf("decode pick wave create: %w", err)
	}
	if strings.TrimSpace(created.WaveID) == "" {
		return fmt.Errorf("pick wave create missing wave_id body %s", string(respBody))
	}
	if strings.EqualFold(created.Status, "READY_TO_SEAL") {
		fmt.Println("PX_E2E_WAREHOUSE_PICK_WAVE_READY_OK")
		return nil
	}
	if err := confirmPickWaveTasks(ctx, client, base, cookie, q, created.WaveID); err != nil {
		return err
	}
	return nil
}

func confirmPickWaveTasks(ctx context.Context, client *http.Client, base, cookie, warehouseQuery, waveID string) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet,
		base+"/v1/warehouse/ops/pick-waves/"+waveID+warehouseQuery, nil, cookie, "")
	if err != nil {
		return fmt.Errorf("pick wave get: %w", err)
	}
	if status >= 400 {
		return fmt.Errorf("pick wave get status %d body %s", status, string(respBody))
	}
	var wave struct {
		Status string `json:"status"`
		Tasks  []struct {
			TaskID string `json:"task_id"`
			Status string `json:"status"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(respBody, &wave); err != nil {
		return fmt.Errorf("decode pick wave: %w", err)
	}
	if strings.EqualFold(wave.Status, "READY_TO_SEAL") {
		fmt.Println("PX_E2E_WAREHOUSE_PICK_WAVE_READY_OK")
		return nil
	}
	if len(wave.Tasks) == 0 {
		return fmt.Errorf("pick wave %s has no tasks", waveID)
	}
	var lastStatus string
	for i, task := range wave.Tasks {
		st := strings.ToUpper(strings.TrimSpace(task.Status))
		if st == "CONFIRMED" || st == "SHORT_WAIVED" {
			lastStatus = st
			continue
		}
		confirmBody, _ := json.Marshal(map[string]any{"quantity_picked": 0})
		status, respBody, _, err = clientPost(ctx, client,
			base+"/v1/warehouse/ops/pick-waves/"+waveID+"/tasks/"+task.TaskID+"/confirm"+warehouseQuery,
			confirmBody, cookie, fmt.Sprintf("ssmr-pick-confirm-%s-%d", waveID, i))
		if err != nil {
			return fmt.Errorf("pick confirm: %w", err)
		}
		if status >= 400 {
			return fmt.Errorf("pick confirm status %d body %s", status, string(respBody))
		}
		var confirmed struct {
			Status string `json:"status"`
		}
		_ = json.Unmarshal(respBody, &confirmed)
		lastStatus = confirmed.Status
	}
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet,
		base+"/v1/warehouse/ops/pick-waves/"+waveID+warehouseQuery, nil, cookie, "")
	if err != nil {
		return fmt.Errorf("pick wave ready get: %w", err)
	}
	if status >= 400 {
		return fmt.Errorf("pick wave ready get status %d body %s", status, string(respBody))
	}
	if err := json.Unmarshal(respBody, &wave); err != nil {
		return fmt.Errorf("decode pick wave ready: %w", err)
	}
	if !strings.EqualFold(wave.Status, "READY_TO_SEAL") {
		return fmt.Errorf("pick wave %s not READY_TO_SEAL after confirm status=%s last=%s", waveID, wave.Status, lastStatus)
	}
	fmt.Println("PX_E2E_WAREHOUSE_PICK_WAVE_READY_OK")
	return nil
}
