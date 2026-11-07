package foundry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	ScenarioOpUpsert = "UPSERT"
	ScenarioOpDelete = "DELETE"
)

// ScenarioObjectPatch represents a Copy-on-Write mutation isolated inside a Scenario sandbox.
type ScenarioObjectPatch struct {
	ScenarioID      string         `json:"scenario_id" spanner:"ScenarioId"`
	ObjectType      string         `json:"object_type" spanner:"ObjectType"`
	ObjectID        string         `json:"object_id" spanner:"ObjectId"`
	SupplierID      string         `json:"supplier_id" spanner:"SupplierId"`
	Operation       string         `json:"operation" spanner:"Operation"` // UPSERT | DELETE
	BaseVersion     int64          `json:"base_version" spanner:"BaseVersion"`
	PatchProperties map[string]any `json:"patch_properties,omitempty" spanner:"-"`
	PatchJson       string         `json:"patch_json,omitempty" spanner:"PatchJson"`
	UpdatedAt       time.Time      `json:"updated_at" spanner:"UpdatedAt"`
	CellID          string         `json:"cell_id" spanner:"-"`
}

type ScenarioPatch = ScenarioObjectPatch
type ScenarioObjectPatches = ScenarioObjectPatch

func (p *ScenarioObjectPatch) PrepareForSpanner() error {
	if p.PatchJson == "" && p.PatchProperties != nil {
		b, err := json.Marshal(p.PatchProperties)
		if err != nil {
			return err
		}
		p.PatchJson = string(b)
	} else if p.PatchJson != "" && p.PatchProperties == nil {
		var props map[string]any
		if err := json.Unmarshal([]byte(p.PatchJson), &props); err == nil {
			p.PatchProperties = props
		}
	}
	return nil
}

// ScenarioWatermarkConflict reports an optimistic concurrency drift between a scenario's
// BaseVersion and the live object's current production Version.
type ScenarioWatermarkConflict struct {
	CellID      string `json:"cell_id"`
	ObjectType  string `json:"object_type"`
	ObjectID    string `json:"object_id"`
	BaseVersion int64  `json:"base_version"`
	LiveVersion int64  `json:"live_version"`
}

// ScenarioCommitResult summarizes a Scenario commit or its watermark conflicts.
type ScenarioCommitResult struct {
	ScenarioID       string                      `json:"scenario_id"`
	CommittedObjects int                         `json:"committed_objects"`
	Conflicts        []ScenarioWatermarkConflict `json:"conflicts,omitempty"`
}

// ApplyScenarioPatch writes a Copy-on-Write patch into the Scenario overlay without mutating live objects.
func (e *Engine) ApplyScenarioPatch(
	ctx context.Context,
	cellID string,
	scenarioID string,
	supplierID string,
	objectType string,
	objectID string,
	operation string,
	patchProps map[string]any,
) (*ScenarioPatch, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cellID = strings.TrimSpace(cellID)
	scenarioID = strings.TrimSpace(scenarioID)
	objectType = strings.TrimSpace(objectType)
	objectID = strings.TrimSpace(objectID)
	operation = strings.ToUpper(strings.TrimSpace(operation))
	if cellID == "" || scenarioID == "" || objectType == "" || objectID == "" {
		return nil, ErrInvalidObjectIdentity
	}
	if operation != ScenarioOpUpsert && operation != ScenarioOpDelete {
		return nil, fmt.Errorf("%w: unsupported scenario operation %q", ErrPreconditionFailed, operation)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	objKey := objectKey(cellID, objectType, objectID)
	var baseVersion int64
	if liveObj, ok := e.objects[objKey]; ok {
		baseVersion = liveObj.Version
	}

	if e.scenarioPatches[scenarioID] == nil {
		e.scenarioPatches[scenarioID] = make(map[string]*ScenarioPatch)
	}

	// Preserve original BaseVersion if the scenario already patched this object once.
	if existingPatch, ok := e.scenarioPatches[scenarioID][objKey]; ok {
		baseVersion = existingPatch.BaseVersion
		merged := cloneMap(existingPatch.PatchProperties)
		for k, v := range patchProps {
			merged[k] = v
		}
		patchProps = merged
	}

	patch := &ScenarioPatch{
		CellID:          cellID,
		ScenarioID:      scenarioID,
		ObjectType:      objectType,
		ObjectID:        objectID,
		SupplierID:      strings.TrimSpace(supplierID),
		Operation:       operation,
		BaseVersion:     baseVersion,
		PatchProperties: cloneMap(patchProps),
		UpdatedAt:       e.now(),
	}
	_ = patch.PrepareForSpanner()
	e.scenarioPatches[scenarioID][objKey] = patch
	return patch, nil
}

// ResolveObjectState computes ResolvedView(Object) = BaseState ⊕ ScenarioPatches - ScenarioDeletions.
func (e *Engine) ResolveObjectState(
	ctx context.Context,
	cellID string,
	scenarioID string,
	objectType string,
	objectID string,
) (*OntologyObject, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	objKey := objectKey(cellID, objectType, objectID)
	baseObj := cloneObject(e.objects[objKey])

	scenarioID = strings.TrimSpace(scenarioID)
	if scenarioID == "" {
		if baseObj == nil {
			return nil, ErrObjectNotFound
		}
		return baseObj, nil
	}

	patches := e.scenarioPatches[scenarioID]
	if patches == nil {
		if baseObj == nil {
			return nil, ErrObjectNotFound
		}
		return baseObj, nil
	}

	patch, hasPatch := patches[objKey]
	if !hasPatch {
		if baseObj == nil {
			return nil, ErrObjectNotFound
		}
		return baseObj, nil
	}

	if patch.Operation == ScenarioOpDelete {
		return nil, ErrObjectNotFound
	}

	if baseObj == nil {
		return &OntologyObject{
			CellID:      strings.TrimSpace(cellID),
			ObjectType:  strings.TrimSpace(objectType),
			ObjectID:    strings.TrimSpace(objectID),
			SupplierID:  patch.SupplierID,
			Version:     1,
			Properties:  cloneMap(patch.PatchProperties),
			ValidFrom:   patch.UpdatedAt,
			UpdatedAt:   patch.UpdatedAt,
		}, nil
	}

	for k, v := range patch.PatchProperties {
		baseObj.Properties[k] = v
	}
	baseObj.UpdatedAt = patch.UpdatedAt
	return baseObj, nil
}

// CommitScenario atomically verifies that every patched object's live production Version matches
// the Scenario's BaseVersion watermark (Guard #7). If any object drifted, it aborts without
// partial writes and returns structured ScenarioWatermarkConflict items.
func (e *Engine) CommitScenario(ctx context.Context, scenarioID string) (*ScenarioCommitResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	scenarioID = strings.TrimSpace(scenarioID)

	e.mu.Lock()
	defer e.mu.Unlock()

	patches := e.scenarioPatches[scenarioID]
	if len(patches) == 0 {
		return &ScenarioCommitResult{ScenarioID: scenarioID, CommittedObjects: 0}, nil
	}

	var conflicts []ScenarioWatermarkConflict
	for objKey, patch := range patches {
		var liveVer int64
		if liveObj, ok := e.objects[objKey]; ok {
			liveVer = liveObj.Version
		}
		if liveVer != patch.BaseVersion {
			conflicts = append(conflicts, ScenarioWatermarkConflict{
				CellID:      patch.CellID,
				ObjectType:  patch.ObjectType,
				ObjectID:    patch.ObjectID,
				BaseVersion: patch.BaseVersion,
				LiveVersion: liveVer,
			})
		}
	}

	if len(conflicts) > 0 {
		return &ScenarioCommitResult{
			ScenarioID: scenarioID,
			Conflicts:  conflicts,
		}, ErrScenarioWatermarkDrift
	}

	committed := 0
	for objKey, patch := range patches {
		if patch.Operation == ScenarioOpDelete {
			delete(e.objects, objKey)
			committed++
			continue
		}
		e.upsertObjectLocked(patch.CellID, patch.ObjectType, patch.ObjectID, patch.SupplierID, patch.PatchProperties, nil)
		committed++
	}

	delete(e.scenarioPatches, scenarioID)
	return &ScenarioCommitResult{
		ScenarioID:       scenarioID,
		CommittedObjects: committed,
	}, nil
}
