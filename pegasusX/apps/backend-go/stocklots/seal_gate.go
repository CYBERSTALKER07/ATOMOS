package stocklots

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
)

var (
	// ErrPickWaveRequired is returned when seal is attempted with no wave (flag on).
	ErrPickWaveRequired = errors.New("pick_wave_required")
	// ErrPickWaveIncomplete is returned when wave exists but is not READY_TO_SEAL.
	ErrPickWaveIncomplete = errors.New("pick_wave_incomplete")
)

// AssertManifestPickReady enforces the Wave 1B seal gate when WMS_PICK_WAVES_ENABLED.
// When WMS_SEAL_SOFT_WARN is on and the wave is incomplete, returns (warn, nil) so seal may proceed.
func AssertManifestPickReady(ctx context.Context, client *spanner.Client, manifestID string) (warn string, err error) {
	if !PickWavesEnabled() {
		return "", nil
	}
	if client == nil {
		return "", fmt.Errorf("%w: spanner unavailable", ErrPickWaveRequired)
	}
	manifestID = strings.TrimSpace(manifestID)
	if manifestID == "" {
		return "", ErrPickWaveRequired
	}
	row, err := client.Single().ReadRow(ctx, "SupplierTruckManifests", spanner.Key{manifestID},
		[]string{"PickWaveId"})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			return "", nil
		}
		return "", err
	}
	var waveID spanner.NullString
	if err := row.Columns(&waveID); err != nil {
		return "", err
	}
	if !waveID.Valid || strings.TrimSpace(waveID.StringVal) == "" {
		if SealSoftWarnEnabled() {
			return "pick_wave_required", nil
		}
		return "", ErrPickWaveRequired
	}
	wRow, err := client.Single().ReadRow(ctx, "PickWaves", spanner.Key{strings.TrimSpace(waveID.StringVal)},
		[]string{"Status"})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			if SealSoftWarnEnabled() {
				return "pick_wave_required", nil
			}
			return "", ErrPickWaveRequired
		}
		return "", err
	}
	var status string
	if err := wRow.Columns(&status); err != nil {
		return "", err
	}
	if !strings.EqualFold(strings.TrimSpace(status), "READY_TO_SEAL") {
		msg := fmt.Sprintf("%s: status=%s", ErrPickWaveIncomplete.Error(), status)
		if SealSoftWarnEnabled() {
			return msg, nil
		}
		return "", fmt.Errorf("%w: status=%s", ErrPickWaveIncomplete, status)
	}
	return "", nil
}
