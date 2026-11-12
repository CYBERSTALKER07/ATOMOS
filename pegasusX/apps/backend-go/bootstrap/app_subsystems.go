package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/catalog"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltower"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditnote"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
	"github.com/pegasusx/pegasusx/apps/backend-go/returns"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
	"github.com/pegasusx/pegasusx/apps/backend-go/segment"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
)

type partnerSubsystems struct {
	svc          *partner.Service
	handlers     *partner.Handlers
	keys         partner.KeyRepository
	jwtSecret    string
	delivery     *partner.DeliveryWorker
	exportWorker *partner.ExportWorker
	ediInbound   *partner.EdiInboundWorker
	ediOutbound  *partner.EdiOutboundWorker
}

func setupPartnerSubsystems(
	spannerClient *spanner.Client,
	cfg *Config,
	orderSvc *order.Service,
	catalogSvc *catalog.Service,
	idemStore idempotency.Store,
	inventorySvc *inventory.Service,
	retailerSvc *retailer.Service,
	log *slog.Logger,
) (*partnerSubsystems, error) {
	var partnerKeys partner.KeyRepository
	var partnerWebhooks partner.WebhookRepository
	var partnerExports partner.ExportRepository
	var partnerSftp partner.SftpConfigRepository
	var partnerEdiDocs partner.EdiDocumentRepository
	if spannerClient != nil {
		partnerKeys = partner.NewSpannerKeyRepository(spannerClient)
		partnerWebhooks = partner.NewSpannerWebhookRepository(spannerClient)
		partnerExports = partner.NewSpannerExportRepository(spannerClient)
		partnerSftp = partner.NewSpannerSftpConfigRepository(spannerClient)
		partnerEdiDocs = partner.NewSpannerEdiDocumentRepository(spannerClient)
	} else {
		if err := cfg.ensureMemoryFallbackAllowed("partner repositories"); err != nil {
			return nil, err
		}
		partnerKeys = partner.NewMemoryKeyRepository()
		partnerWebhooks = partner.NewMemoryWebhookRepository()
		partnerExports = partner.NewMemoryExportRepository()
		partnerSftp = partner.NewMemorySftpConfigRepository()
		partnerEdiDocs = partner.NewMemoryEdiDocumentRepository()
	}
	partnerSvc := partner.NewService(partnerKeys, partnerWebhooks, orderSvc, catalogSvc, log)
	partnerSvc.SetExportRepos(partnerExports, partnerSftp)
	partnerSvc.SetIdempotencyStore(idemStore)
	partnerSvc.SetInventoryService(inventorySvc)
	if spannerClient != nil {
		partnerSvc.SetPOSFeedSink(partner.NewSpannerPOSFeedSink(spannerClient))
	}
	var partnerAs2 partner.As2ConfigRepository
	var partnerCoa partner.CoaRepository
	var partnerEdiProfiles partner.EdiProfileRepository
	if spannerClient != nil {
		partnerAs2 = partner.NewSpannerAs2ConfigRepository(spannerClient)
		partnerCoa = partner.NewSpannerCoaRepository(spannerClient)
		partnerEdiProfiles = &partner.SpannerEdiProfiles{Client: spannerClient}
	} else {
		if err := cfg.ensureMemoryFallbackAllowed("partner AS2/CoA/EDI repositories"); err != nil {
			return nil, err
		}
		partnerAs2 = partner.NewMemoryAs2ConfigRepository()
		partnerCoa = partner.NewMemoryCoaRepository()
		partnerEdiProfiles = partner.NewMemoryEdiProfiles()
	}
	partnerSvc.SetAs2Repository(partnerAs2)
	partnerSvc.SetCoaRepository(partnerCoa)
	partnerDelivery := partner.NewDeliveryWorker(partnerWebhooks, log)
	partnerExportWorker := partner.NewExportWorker(partnerExports, partnerSftp, spannerClient, log)
	partnerExportWorker.SetCoaRepository(partnerCoa)
	partnerEdiOut := partner.NewEdiOutboundWorker(partnerEdiDocs, partnerSftp, orderSvc, spannerClient, log)
	partnerEdiOut.SetAs2Repository(partnerAs2)
	partnerSvc.SetEdiRepos(partnerEdiDocs, partnerEdiOut)
	partnerEdiIn := partner.NewEdiInboundWorker(partnerEdiDocs, partnerSftp, partnerSvc, log)
	partnerSvc.SetEdiProfiles(partnerEdiProfiles)
	partnerEdiIn.SetEdiProfiles(partnerEdiProfiles)
	partnerEdiOut.SetEdiProfiles(partnerEdiProfiles)
	partnerEdiIn.SetAckEnqueuer(partnerEdiOut)
	partnerEdiIn.ResolveGeo = func(ctx context.Context, retailerID string) (partner.RetailerGeo, error) {
		loc, err := retailerSvc.EnsurePrimaryLocation(ctx, retailerID)
		if err != nil {
			return partner.RetailerGeo{}, err
		}
		return partner.RetailerGeo{Lat: loc.Lat, Lng: loc.Lng, H3Cell: loc.H3Cell}, nil
	}
	partnerHandlers := &partner.Handlers{Svc: partnerSvc, Delivery: partnerDelivery, EdiInbound: partnerEdiIn}
	partnerJWTSecret := partner.ResolvePartnerJWTSecret(cfg.JWTSecret)
	partnerSvc.SetOAuthJWT(partnerJWTSecret, partner.PartnerOAuthIssuer(), partner.PartnerOAuthTTL())

	return &partnerSubsystems{
		svc:          partnerSvc,
		handlers:     partnerHandlers,
		keys:         partnerKeys,
		jwtSecret:    partnerJWTSecret,
		delivery:     partnerDelivery,
		exportWorker: partnerExportWorker,
		ediInbound:   partnerEdiIn,
		ediOutbound:  partnerEdiOut,
	}, nil
}

func setupControlTowerSubsystem(
	spannerClient *spanner.Client,
	cacheClient *cache.Cache,
	creditNoteSvc *creditnote.Service,
	creditSvc *credit.Service,
	routingSvc *routing.Service,
	notifSvc *notifications.Service,
	returnsSvc *returns.Service,
	supplierSvc *supplier.Service,
	replenishmentEngine *replenishment.Engine,
	log *slog.Logger,
) (*controltower.Service, *controltower.Handlers, *controltower.Worker) {
	if spannerClient == nil {
		return nil, nil, nil
	}
	ctCfg := controltower.ConfigFromEnv()
	ctRepo := controltower.NewSpannerRepository(spannerClient)
	cnSvc := creditNoteSvc
	if cnSvc == nil {
		cnSvc = creditnote.NewService(creditnote.NewSpannerRepository(spannerClient))
	}
	planSvc := planning.NewService(spannerClient).WithCache(cacheClient)
	planSvc.TwinScenarioEnabled = strings.EqualFold(strings.TrimSpace(os.Getenv("TWIN_SCENARIO_ENABLED")), "true")
	segSvc := segment.NewService(segment.NewSpannerRepository(spannerClient))
	ctExecutor := controltower.NewActionExecutor(ctRepo, controltower.ExecutorDeps{
		CreditNotes:   cnSvc,
		Credit:        creditSvc,
		Planning:      planSvc,
		Routing:       routingSvc,
		Notifications: notifSvc,
		Returns:       returnsSvc,
		Segment:       segSvc,
		Log:           log,
	})
	ctEngine := controltower.NewEngine(ctRepo, ctExecutor, segSvc, ctCfg, log)
	controlTowerSvc := controltower.NewService(ctRepo, ctEngine, ctCfg)
	controlTowerHandlers := controltower.NewHandlers(controlTowerSvc, supplierSvc.ScopedSupplierID)
	controlTowerWorker := controltower.NewWorker(controlTowerSvc, log, 3*time.Minute)
	if ctCfg.Enabled {
		log.Info("control tower playbooks enabled", "auto_execute", ctCfg.AutoExecute)
	}
	supplierSvc.SetControlTower(controlTowerSvc)
	if replenishmentEngine != nil {
		replenishmentEngine.SegmentSvc = segSvc
	}
	return controlTowerSvc, controlTowerHandlers, controlTowerWorker
}
