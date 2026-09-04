package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/claims"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
	"github.com/pegasusx/pegasusx/apps/backend-go/returns"
	"github.com/pegasusx/pegasusx/apps/backend-go/twin"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

type kafkaConsumers struct {
	notificationConsumer *kafka.Consumer
	orderEventConsumer    *kafka.Consumer
	warehouseEventConsumer *kafka.Consumer
	returnsEventConsumer  *kafka.Consumer
	claimsEventConsumer   *kafka.Consumer
	billingTierConsumer   *kafka.Consumer
	partnerEventConsumer  *kafka.Consumer
	twinEventConsumer     *kafka.Consumer
	cleanup               func()
}

// setupKafkaConsumers initializes all domain event consumers.
func setupKafkaConsumers(
	cfg *Config,
	kafkaAuth kafkautil.ClientAuth,
	redisAdapter redisRuntimeAdapter,
	spannerClient *spanner.Client,
	cacheClient *cache.Cache,
	pushBridge *notifications.PushBridge,
	notifSvc *notifications.Service,
	retailerSvc *retailer.Service,
	retailerHub *ws.Hub,
	supplierHub *ws.Hub,
	driverHub *ws.Hub,
	warehouseHub *ws.Hub,
	factoryHub *ws.Hub,
	payloadHub *ws.Hub,
	orderSvc *order.Service,
	warehouseSvc *warehouse.Service,
	returnsSvc *returns.Service,
	claimsSvc *claims.Service,
	partnerSvc *partner.Service,
	fxSvc *fxrates.Service,
	log *slog.Logger,
) (*kafkaConsumers, error) {
	dlqWriter, err := newKafkaRuntimeDLQWriter(cfg.KafkaBrokers, cfg.KafkaTopicMainDLQ, kafkaAuth)
	if err != nil {
		if cfg.RequireInfraAdapters {
			return nil, fmt.Errorf("init notification dlq writer: %w", err)
		}
		log.Warn("notification consumer disabled; dlq writer init failed",
			"topic", cfg.KafkaTopicMain,
			"dlq_topic", cfg.KafkaTopicMainDLQ,
			"err", err,
		)
		return &kafkaConsumers{}, nil
	}

	var kafkaEventDedup kafka.EventDedupStore = kafka.NewInMemoryEventDedup(7 * 24 * time.Hour)
	if spannerClient != nil {
		kafkaEventDedup = kafka.NewSpannerEventDedup(spannerClient)
	} else if redisAdapter != nil {
		if rc := redisAdapter.Client(); rc != nil {
			kafkaEventDedup = kafka.NewRedisEventDedup(rc, 7*24*time.Hour)
		}
	}
	const notificationConsumerGroup = "void-notification-dispatcher"
	const orderConsumerGroup = "void-order-mutator"
	const warehouseConsumerGroup = "void-warehouse-mutator"
	dispatcher := kafka.NewNotificationDispatcher(kafka.DispatcherDeps{
		RetailerHub:     retailerHub,
		SupplierHub:     supplierHub,
		DriverHub:       driverHub,
		WarehouseHub:    warehouseHub,
		FactoryHub:      factoryHub,
		PayloadHub:      payloadHub,
		Push:            pushBridge,
		Inbox:           notifSvc,
		EventDedup:      kafkaEventDedup,
		ConsumerGroupID: notificationConsumerGroup,
		Cache:           cacheClient,
		RetailerActors:  retailerSvc, // Phase 1 multi-user FCM fanout
	})
	dispatcherTopics := events.DispatcherConsumerTopics()
	notificationConsumer := kafka.NewMultiTopicConsumer(kafka.ConsumerDeps{
		Brokers:   strings.Split(cfg.KafkaBrokers, ","),
		GroupID:   notificationConsumerGroup,
		Topics:    dispatcherTopics,
		Handler:   dispatcher.HandleEvent,
		DLQWriter: dlqWriter,
		Auth:      kafkaAuth,
	})
	orderHandler := kafka.WithEventDedup(kafkaEventDedup, orderConsumerGroup, order.NewEventConsumer(orderSvc, log).HandleEvent)
	warehouseHandler := kafka.WithEventDedup(kafkaEventDedup, warehouseConsumerGroup, warehouse.NewEventConsumer(warehouseSvc, log).HandleEvent)
	const returnsConsumerGroup = "void-returns-reverse"
	returnsHandler := kafka.WithEventDedup(kafkaEventDedup, returnsConsumerGroup, returns.NewEventConsumer(returnsSvc, log).HandleEvent)
	orderEventConsumer := kafka.NewConsumer(kafka.ConsumerDeps{
		Brokers:   strings.Split(cfg.KafkaBrokers, ","),
		GroupID:   orderConsumerGroup,
		Topic:     events.OrderConsumerTopic(),
		Handler:   orderHandler,
		DLQWriter: dlqWriter,
		Auth:      kafkaAuth,
	})
	warehouseEventConsumer := kafka.NewConsumer(kafka.ConsumerDeps{
		Brokers:   strings.Split(cfg.KafkaBrokers, ","),
		GroupID:   warehouseConsumerGroup,
		Topic:     events.DispatchConsumerTopic(),
		Handler:   warehouseHandler,
		DLQWriter: dlqWriter,
		Auth:      kafkaAuth,
	})
	returnsEventConsumer := kafka.NewConsumer(kafka.ConsumerDeps{
		Brokers:   strings.Split(cfg.KafkaBrokers, ","),
		GroupID:   returnsConsumerGroup,
		Topic:     events.TopicExceptions,
		Handler:   returnsHandler,
		DLQWriter: dlqWriter,
		Auth:      kafkaAuth,
	})

	const claimsConsumerGroup = "void-claims-bridge"
	claimsHandler := kafka.WithEventDedup(kafkaEventDedup, claimsConsumerGroup, claims.NewQuarantineBridge(claimsSvc, log).HandleEvent)
	claimsEventConsumer := kafka.NewConsumer(kafka.ConsumerDeps{
		Brokers:   strings.Split(cfg.KafkaBrokers, ","),
		GroupID:   claimsConsumerGroup,
		Topic:     events.TopicMain,
		Handler:   claimsHandler,
		DLQWriter: dlqWriter,
		Auth:      kafkaAuth,
	})

	var billingTierConsumer *kafka.Consumer
	if spannerClient != nil {
		const billingConsumerGroup = "void-billing-tier"
		meterWorker := billing.NewMeterWorker(spannerClient)
		billingWorker := kafka.NewBillingTierWorker(meterWorker).WithFx(fxSvc, cfg.SeedSupplierCurrency)
		billingHandler := kafka.WithEventDedup(kafkaEventDedup, billingConsumerGroup, billingWorker.HandleEvent)
		billingTierConsumer = kafka.NewConsumer(kafka.ConsumerDeps{
			Brokers:   strings.Split(cfg.KafkaBrokers, ","),
			GroupID:   billingConsumerGroup,
			Topic:     events.OrderConsumerTopic(),
			Handler:   billingHandler,
			DLQWriter: dlqWriter,
			Auth:      kafkaAuth,
		})
	}
	const partnerWebhookConsumerGroup = "void-partner-webhooks"
	partnerHandler := kafka.WithEventDedup(kafkaEventDedup, partnerWebhookConsumerGroup, partner.NewEventConsumer(partnerSvc, log).HandleEvent)
	partnerEventConsumer := kafka.NewMultiTopicConsumer(kafka.ConsumerDeps{
		Brokers:   strings.Split(cfg.KafkaBrokers, ","),
		GroupID:   partnerWebhookConsumerGroup,
		Topics:    []string{events.OrderConsumerTopic(), events.TopicExceptions},
		Handler:   partnerHandler,
		DLQWriter: dlqWriter,
		Auth:      kafkaAuth,
	})

	var twinEventConsumer *kafka.Consumer
	if spannerClient != nil {
		const twinConsumerGroup = "void-digital-twin"
		twinSvc := twin.NewService(twin.ServiceConfig{
			Repo: twin.NewSpannerRepository(spannerClient),
			Log:  log,
		})
		twinHandler := kafka.WithEventDedup(kafkaEventDedup, twinConsumerGroup, twin.NewEventConsumer(twinSvc, log).HandleEvent)
		twinEventConsumer = kafka.NewMultiTopicConsumer(kafka.ConsumerDeps{
			Brokers:   strings.Split(cfg.KafkaBrokers, ","),
			GroupID:   twinConsumerGroup,
			Topics:    events.TwinConsumerTopics(),
			Handler:   twinHandler,
			DLQWriter: dlqWriter,
			Auth:      kafkaAuth,
		})
	}

	cleanup := func() {
		if err := notificationConsumer.Close(); err != nil {
			log.Warn("notification consumer close failed", "err", err)
		}
		if err := orderEventConsumer.Close(); err != nil {
			log.Warn("order event consumer close failed", "err", err)
		}
		if partnerEventConsumer != nil {
			if err := partnerEventConsumer.Close(); err != nil {
				log.Warn("partner webhook consumer close failed", "err", err)
			}
		}
		if twinEventConsumer != nil {
			if err := twinEventConsumer.Close(); err != nil {
				log.Warn("twin event consumer close failed", "err", err)
			}
		}
		if err := warehouseEventConsumer.Close(); err != nil {
			log.Warn("warehouse event consumer close failed", "err", err)
		}
		if returnsEventConsumer != nil {
			if err := returnsEventConsumer.Close(); err != nil {
				log.Warn("returns event consumer close failed", "err", err)
			}
		}
		if billingTierConsumer != nil {
			if err := billingTierConsumer.Close(); err != nil {
				log.Warn("billing tier consumer close failed", "err", err)
			}
		}
		if err := dlqWriter.Close(); err != nil {
			log.Warn("notification dlq writer close failed", "err", err)
		}
	}

	log.Info("notification consumer enabled",
		"topic", cfg.KafkaTopicMain,
		"order_topic", events.OrderConsumerTopic(),
		"dispatch_topic", events.DispatchConsumerTopic(),
		"exceptions_topic", events.TopicExceptions,
		"dlq_topic", cfg.KafkaTopicMainDLQ,
		"consume_domain", events.ConsumeDomainTopics(),
		"dual_write", events.DualWriteDomainTopics(),
		"billing_tier", billingTierConsumer != nil,
		"returns_reverse", returnsEventConsumer != nil,
		"digital_twin", twinEventConsumer != nil,
	)

	return &kafkaConsumers{
		notificationConsumer:   notificationConsumer,
		orderEventConsumer:     orderEventConsumer,
		warehouseEventConsumer: warehouseEventConsumer,
		returnsEventConsumer:   returnsEventConsumer,
		claimsEventConsumer:    claimsEventConsumer,
		billingTierConsumer:    billingTierConsumer,
		partnerEventConsumer:   partnerEventConsumer,
		twinEventConsumer:      twinEventConsumer,
		cleanup:                cleanup,
	}, nil
}

// setupDunningNotification wires in-app and multi-channel off-app dunning notifications.
func setupDunningNotification(
	spannerClient *spanner.Client,
	notifSvc *notifications.Service,
	pushBridge *notifications.PushBridge,
	log *slog.Logger,
) (func(ctx context.Context, inv ar.Invoice, prevStep, nextStep int64) error, error) {
	inAppNotify := func(ctx context.Context, inv ar.Invoice, prevStep, nextStep int64) error {
		eventType, title, body := ar.NotifyMessage(inv, nextStep)
		deep := "/credit/invoices/" + inv.InvoiceID
		if notifSvc != nil {
			_ = notifSvc.CreateNotification(ctx, inv.RetailerID, "RETAILER", eventType, title, body, deep)
			_ = notifSvc.CreateNotification(ctx, inv.SupplierID, "ADMIN", eventType, title, body, "/credit/collections")
		}
		if pushBridge != nil {
			data := map[string]string{
				"type": eventType, "invoice_id": inv.InvoiceID, "order_id": inv.OrderID,
				"step": ar.StepName(nextStep), "prev_step": ar.StepName(prevStep),
			}
			pushBridge.NotifyActor(ctx, inv.RetailerID, "RETAILER", data)
			pushBridge.NotifyActor(ctx, inv.SupplierID, "ADMIN", data)
		}
		return nil
	}

	dunningTransports, err := ar.TransportsFromEnv()
	if err != nil {
		return nil, fmt.Errorf("dunning transports: %w", err)
	}
	offAppNotify := ar.MultiChannelNotify(log, ar.NewSpannerContactResolver(spannerClient), dunningTransports)

	return func(ctx context.Context, inv ar.Invoice, prevStep, nextStep int64) error {
		inAppErr := inAppNotify(ctx, inv, prevStep, nextStep)
		offAppErr := offAppNotify(ctx, inv, prevStep, nextStep)
		return errors.Join(inAppErr, offAppErr)
	}, nil
}
