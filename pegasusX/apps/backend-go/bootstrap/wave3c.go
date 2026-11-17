// Package bootstrap wires ATOMOS Wave 3C services (M1.9 connectors, M1.10 exportpacks,
// time-boxed AUDITOR IAM JWT configuration, audit hooks, and ontology lineage propagation).
package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/connectors"
	"github.com/pegasusx/pegasusx/apps/backend-go/exportpacks"
	"github.com/pegasusx/pegasusx/apps/backend-go/ontology"
)

type connectorsAuditAdapter struct {
	auditSvc *audit.Service
}

func (a *connectorsAuditAdapter) RecordEvent(ctx context.Context, ev connectors.AuditRecordInput) error {
	if a == nil || a.auditSvc == nil {
		return nil
	}
	_, err := a.auditSvc.Record(ctx, audit.EventInput{
		TenantID:   ev.TenantID,
		ActorID:    ev.ActorID,
		ActorType:  audit.NormalizeActorType(ev.ActorType),
		Role:       ev.Role,
		Action:     ev.Action,
		ObjectType: ev.ObjectType,
		ObjectID:   ev.ObjectID,
		BeforeJSON: ev.BeforeJSON,
		AfterJSON:  ev.AfterJSON,
		ReasonCode: ev.ReasonCode,
		TraceID:    ev.TraceID,
		IP:         ev.IP,
		DeviceID:   ev.DeviceID,
	})
	return err
}

// NewConnectorsAuditRecorder wraps audit.Service into connectors.AuditRecorder.
func NewConnectorsAuditRecorder(auditSvc *audit.Service) connectors.AuditRecorder {
	if auditSvc == nil {
		return nil
	}
	return &connectorsAuditAdapter{auditSvc: auditSvc}
}

type exportPacksAuditAdapter struct {
	auditSvc *audit.Service
}

func toExportAuditEventInput(ev exportpacks.AuditRecordInput) audit.EventInput {
	return audit.EventInput{
		TenantID:   ev.TenantID,
		ActorID:    ev.ActorID,
		ActorType:  audit.NormalizeActorType(ev.ActorType),
		Role:       ev.Role,
		Action:     ev.Action,
		ObjectType: ev.ObjectType,
		ObjectID:   ev.ObjectID,
		BeforeJSON: ev.BeforeJSON,
		AfterJSON:  ev.AfterJSON,
		ReasonCode: ev.ReasonCode,
		TraceID:    ev.TraceID,
		IP:         ev.IP,
		DeviceID:   ev.DeviceID,
	}
}

func (a *exportPacksAuditAdapter) RecordEvent(ctx context.Context, ev exportpacks.AuditRecordInput) error {
	if a == nil || a.auditSvc == nil {
		return nil
	}
	_, err := a.auditSvc.Record(ctx, toExportAuditEventInput(ev))
	return err
}

func (a *exportPacksAuditAdapter) RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev exportpacks.AuditRecordInput) error {
	if a == nil || a.auditSvc == nil || txn == nil {
		return nil
	}
	_, err := a.auditSvc.RecordInTxn(ctx, txn, toExportAuditEventInput(ev))
	return err
}

// NewExportPacksAuditRecorder wraps audit.Service into exportpacks.AuditRecorder (standalone and transactional).
func NewExportPacksAuditRecorder(auditSvc *audit.Service) exportpacks.AuditRecorder {
	if auditSvc == nil {
		return nil
	}
	return &exportPacksAuditAdapter{auditSvc: auditSvc}
}

func mapConnectorObjectTypeToOntology(connectorObjType string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(connectorObjType)) {
	case strings.ToLower(connectors.ObjectTypeProduct), "sku":
		return ontology.TypeSKU, true
	case strings.ToLower(connectors.ObjectTypeOrder):
		return ontology.TypeOrder, true
	case strings.ToLower(connectors.ObjectTypeInvoice), "invoice":
		return ontology.TypeNasiyaInvoice, true
	case strings.ToLower(connectors.ObjectTypeShop), strings.ToLower(connectors.ObjectTypeClient), "retailer":
		return ontology.TypeShop, true
	case "warehouse":
		return ontology.TypeWarehouse, true
	default:
		return "", false
	}
}

// BuildConnectorOntologyLineageHook annotates existing OntologyObjects with external source_system
// and external_id lineage whenever connector shadow-compare or pull matches internal objects.
func BuildConnectorOntologyLineageHook(ontSvc *ontology.Service) connectors.OntologyLineageHook {
	if ontSvc == nil || ontSvc.Repository() == nil {
		return nil
	}
	return func(ctx context.Context, in connectors.OntologyLineageInput) error {
		if ontSvc == nil || ontSvc.Repository() == nil || len(in.EntityMaps) == 0 {
			return nil
		}
		repo := ontSvc.Repository()
		type dedupKey struct {
			objType string
			objID   string
		}
		seen := make(map[dedupKey]struct{}, len(in.EntityMaps))
		updates := make([]*ontology.OntologyObject, 0, len(in.EntityMaps))
		syncedAt := in.SyncedAt.UTC()
		if syncedAt.IsZero() {
			syncedAt = time.Now().UTC()
		}

		for _, m := range in.EntityMaps {
			internalID := strings.TrimSpace(m.InternalID)
			if internalID == "" {
				continue
			}
			ontType, ok := mapConnectorObjectTypeToOntology(m.ObjectType)
			if !ok {
				continue
			}
			dk := dedupKey{objType: ontType, objID: internalID}
			if _, dup := seen[dk]; dup {
				continue
			}
			seen[dk] = struct{}{}

			existing, err := repo.GetObject(ctx, ontology.DefaultCellID, ontType, internalID)
			if err != nil || existing == nil {
				continue
			}
			props := make(map[string]any, len(existing.Properties)+4)
			for k, v := range existing.Properties {
				props[k] = v
			}
			props["source_system"] = in.Provider
			props["external_id"] = m.ExternalID
			if in.RunID != "" {
				props["connector_run_id"] = in.RunID
			}
			props["connector_synced_at"] = syncedAt.Format(time.RFC3339)

			supID := strings.TrimSpace(existing.SupplierID)
			if supID == "" {
				supID = strings.TrimSpace(in.TenantID)
			}
			ver := existing.Version + 1
			if ver < 1 {
				ver = 1
			}
			updates = append(updates, &ontology.OntologyObject{
				CellID:           existing.CellID,
				ObjectType:       existing.ObjectType,
				ObjectID:         existing.ObjectID,
				SupplierID:       supID,
				Version:          ver,
				Properties:       props,
				SecurityMarkings: append([]string(nil), existing.SecurityMarkings...),
				ValidFrom:        existing.ValidFrom,
			})
		}

		if len(updates) == 0 {
			return nil
		}
		_, err := ontSvc.UpsertDirect(ctx, updates, nil, ontology.UpsertOptions{
			StrictMonotonic:     true,
			EmitOutbox:          true,
			SkipNoOpIfUnchanged: true,
		})
		return err
	}
}

// SetupWave3C initializes Wave 3C connector spine, export packs, audit recorders,
// ontology lineage hook, and time-boxed AUDITOR IAM JWT signing on app.
func SetupWave3C(_ context.Context, cfg *Config, spannerClient *spanner.Client, app *App, log *slog.Logger) error {
	if app == nil {
		return nil
	}

	if spannerClient != nil {
		if app.Connectors == nil {
			app.Connectors = connectors.NewService(spannerClient)
		}
		if app.ExportPacks == nil {
			app.ExportPacks = exportpacks.NewService(spannerClient)
		}
	} else if cfg != nil && cfg.AllowMemoryFallback {
		if app.Connectors == nil {
			app.Connectors = connectors.NewServiceWithRepository(connectors.NewMemoryRepository())
		}
		if app.ExportPacks == nil {
			app.ExportPacks = exportpacks.NewServiceWithDeps(
				exportpacks.NewMemoryRepository(),
				&exportpacks.MemoryCollector{},
				exportpacks.NewMemoryPackStorage(),
			)
		}
	}

	if app.Connectors != nil {
		if app.Audit != nil {
			app.Connectors.SetAuditRecorder(NewConnectorsAuditRecorder(app.Audit))
		}
		if app.Ontology != nil {
			app.Connectors.SetOntologyLineageHook(BuildConnectorOntologyLineageHook(app.Ontology))
		}
	}

	if app.ExportPacks != nil {
		if app.Audit != nil {
			app.ExportPacks.SetAuditRecorder(NewExportPacksAuditRecorder(app.Audit))
		}
		signingSecret := strings.TrimSpace(os.Getenv("EXPORT_PACKS_SIGNING_SECRET"))
		if signingSecret == "" && cfg != nil {
			signingSecret = strings.TrimSpace(cfg.JWTSecret)
		}
		if signingSecret != "" {
			app.ExportPacks.SetSigningSecret(signingSecret)
		}
	}

	if app.IAM != nil && cfg != nil && strings.TrimSpace(cfg.JWTSecret) != "" {
		app.IAM.SetJWTSecret(cfg.JWTSecret, cfg.JWTIssuer)
	}

	if log != nil {
		log.Info("wave3c initialized",
			"connectors_enabled", app.Connectors != nil,
			"exportpacks_enabled", app.ExportPacks != nil,
		)
	}
	return nil
}
