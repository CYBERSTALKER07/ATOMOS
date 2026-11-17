package events

import (
	"os"
	"testing"
)

func TestDomainTopicForEventType(t *testing.T) {
	t.Parallel()
	if got := DomainTopicForEventType(EventOrderCreated); got != TopicOrders {
		t.Fatalf("ORDER_CREATED -> %q, want %q", got, TopicOrders)
	}
	if got := DomainTopicForEventType(EventManifestSealed); got != TopicDispatch {
		t.Fatalf("MANIFEST_SEALED -> %q, want %q", got, TopicDispatch)
	}
	if got := DomainTopicForEventType(EventDriverLocationUpdated); got != TopicRealtime {
		t.Fatalf("DRIVER_LOCATION -> %q, want %q", got, TopicRealtime)
	}
	if got := DomainTopicForEventType(EventSupplierCreated); got != "" {
		t.Fatalf("SUPPLIER_CREATED -> %q, want empty", got)
	}
	if got := DomainTopicForEventType(EventDemandSignal); got != TopicDemand {
		t.Fatalf("DEMAND_SIGNAL -> %q, want %q", got, TopicDemand)
	}
	if got := DomainTopicForEventType(EventCashierSessionTransition); got != TopicOrders {
		t.Fatalf("CASHIER_SESSION_TRANSITION -> %q, want %q", got, TopicOrders)
	}
	if got := DomainTopicForEventType(EventFleetDVIRRedTag); got != TopicDispatch {
		t.Fatalf("FLEET_DVIR_RED_TAG -> %q, want %q", got, TopicDispatch)
	}
	if got := DomainTopicForEventType(EventYardGateCheckRecorded); got != TopicDispatch {
		t.Fatalf("YARD_GATE_CHECK_RECORDED -> %q, want %q", got, TopicDispatch)
	}
	if got := DomainTopicForEventType(EventContainerLedgerUpdated); got != TopicExceptions {
		t.Fatalf("CONTAINER_LEDGER_UPDATED -> %q, want %q", got, TopicExceptions)
	}
	if got := DomainTopicForEventType(EventGLEntryPosted); got != TopicLedger {
		t.Fatalf("gl.entry.posted -> %q, want %q", got, TopicLedger)
	}
	if got := DomainTopicForEventType(EventGLEntryReversed); got != TopicLedger {
		t.Fatalf("gl.entry.reversed -> %q, want %q", got, TopicLedger)
	}
	for _, ev := range []string{
		EventReconRunCompleted, EventReconBreakOpened, EventReconBreakResolved,
		EventAuditAnchorWritten, EventIAMRoleChanged, EventSodViolationDetected,
	} {
		if got := DomainTopicForEventType(ev); got != TopicLedger {
			t.Fatalf("%s -> %q, want %q", ev, got, TopicLedger)
		}
	}
	if got := DomainTopicForEventType(EventOntologyObjectChanged); got != TopicIntel {
		t.Fatalf("%s -> %q, want %q", EventOntologyObjectChanged, got, TopicIntel)
	}
	if got := DomainTopicForEventType(EventOntologyObjectUpdated); got != TopicDispatch {
		t.Fatalf("%s -> %q, want %q", EventOntologyObjectUpdated, got, TopicDispatch)
	}
	for _, ev := range []string{
		EventPeriodSoftClosed, EventPeriodClosed, EventPeriodReopened,
		EventEsfSubmitted, EventEsfAccepted, EventEsfRejected, EventEsfCancelled,
	} {
		if got := DomainTopicForEventType(ev); got != TopicLedger {
			t.Fatalf("%s -> %q, want %q", ev, got, TopicLedger)
		}
	}
	for _, ev := range []string{
		EventApprovalRequested, EventApprovalDecided,
	} {
		if got := DomainTopicForEventType(ev); got != TopicMain {
			t.Fatalf("%s -> %q, want %q", ev, got, TopicMain)
		}
	}
	for _, ev := range []string{
		EventInventoryReceived, EventInventoryIssued, EventInventoryExpiryRisk,
		EventInventoryRebalanceProposed,
	} {
		if got := DomainTopicForEventType(ev); got != TopicDispatch {
			t.Fatalf("%s -> %q, want %q", ev, got, TopicDispatch)
		}
	}
	for _, ev := range []string{
		EventCreditLimitChanged, EventCreditHoldApplied, EventCreditHoldReleased,
	} {
		if got := DomainTopicForEventType(ev); got != TopicMain {
			t.Fatalf("%s -> %q, want %q", ev, got, TopicMain)
		}
	}
	for _, ev := range []string{
		EventForecastRunCompleted, EventCreditScoreComputed,
		EventProposalCreated, EventProposalDecided, EventProposalExecuted,
		EventRoiEventRecorded,
	} {
		if got := DomainTopicForEventType(ev); got != TopicIntel {
			t.Fatalf("%s -> %q, want %q", ev, got, TopicIntel)
		}
	}
	for _, ev := range []string{
		EventConnectorSyncCompleted, EventConnectorSyncFailed, EventConnectorShadowMismatch,
	} {
		if got := DomainTopicForEventType(ev); got != TopicConnectors {
			t.Fatalf("%s -> %q, want %q", ev, got, TopicConnectors)
		}
	}
	for _, ev := range []string{
		EventExportPackCreated, EventAuditorAccessGranted,
		EventEntityIntercompanyPosted, EventFxRevaluationPosted, EventP2PThreeWayMatchCompleted,
	} {
		if got := DomainTopicForEventType(ev); got != TopicLedger {
			t.Fatalf("%s -> %q, want %q", ev, got, TopicLedger)
		}
	}
	if got := DomainTopicForEventType(EventP2PPOCreated); got != TopicOrders {
		t.Fatalf("%s -> %q, want %q", EventP2PPOCreated, got, TopicOrders)
	}
	if got := DomainTopicForEventType(EventP2PReceiptPosted); got != TopicDispatch {
		t.Fatalf("%s -> %q, want %q", EventP2PReceiptPosted, got, TopicDispatch)
	}
	if got := DomainTopicForEventType(EventConnectorPushCompleted); got != TopicConnectors {
		t.Fatalf("%s -> %q, want %q", EventConnectorPushCompleted, got, TopicConnectors)
	}
	for _, ev := range []string{EventAnomalyDetected, EventAnomalyResolved} {
		if got := DomainTopicForEventType(ev); got != TopicExceptions {
			t.Fatalf("%s -> %q, want %q", ev, got, TopicExceptions)
		}
	}
	if got := DomainTopicForEventType(EventWhatIfSimulated); got != TopicIntel {
		t.Fatalf("%s -> %q, want %q", EventWhatIfSimulated, got, TopicIntel)
	}
	if got := DomainTopicForEventType(EventHaccpExcursionReported); got != TopicTelemetryLogistics {
		t.Fatalf("%s -> %q, want %q", EventHaccpExcursionReported, got, TopicTelemetryLogistics)
	}
	if got := DomainTopicForEventType(EventProposalL2AutoExecuted); got != TopicMain {
		t.Fatalf("%s -> %q, want %q", EventProposalL2AutoExecuted, got, TopicMain)
	}
	if got := DomainTopicForEventType(EventScimIdentitySynced); got != TopicLedger {
		t.Fatalf("%s -> %q, want %q", EventScimIdentitySynced, got, TopicLedger)
	}
}

func TestTopicIntelAndConnectors_DefaultAndEnvOverride(t *testing.T) {
	if TopicIntel != "pegasusx-intel" {
		t.Fatalf("TopicIntel = %q, want %q", TopicIntel, "pegasusx-intel")
	}
	if TopicConnectors != "pegasusx-connectors" {
		t.Fatalf("TopicConnectors = %q, want %q", TopicConnectors, "pegasusx-connectors")
	}
	t.Setenv("KAFKA_TOPIC_INTEL", "custom-intel-topic")
	if got := topicFromEnv("KAFKA_TOPIC_INTEL", "pegasusx-intel"); got != "custom-intel-topic" {
		t.Fatalf("topicFromEnv intel = %q, want %q", got, "custom-intel-topic")
	}
	t.Setenv("KAFKA_TOPIC_CONNECTORS", "custom-connectors-topic")
	if got := topicFromEnv("KAFKA_TOPIC_CONNECTORS", "pegasusx-connectors"); got != "custom-connectors-topic" {
		t.Fatalf("topicFromEnv connectors = %q, want %q", got, "custom-connectors-topic")
	}
}

func TestRelayPublishTopics_intelDualWrite(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_DUAL_WRITE", "true")
	payload := []byte(`{"type":"ontology.object.changed","object_type":"Order","object_id":"ord-1"}`)
	if topics := RelayPublishTopics(TopicMain, payload); len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicIntel {
		t.Fatalf("RelayPublishTopics(TopicMain, ontology.object.changed) = %v, want [%s %s]", topics, TopicMain, TopicIntel)
	}
	if topics := RelayPublishTopics(TopicIntel, payload); len(topics) != 1 || topics[0] != TopicIntel {
		t.Fatalf("RelayPublishTopics(TopicIntel, ontology.object.changed) = %v, want [%s]", topics, TopicIntel)
	}
}

func TestTopicLedger_DefaultAndEnvOverride(t *testing.T) {
	if TopicLedger != "pegasusx-ledger" {
		t.Fatalf("TopicLedger = %q, want %q", TopicLedger, "pegasusx-ledger")
	}
	t.Setenv("KAFKA_TOPIC_LEDGER", "custom-ledger-topic")
	if got := topicFromEnv("KAFKA_TOPIC_LEDGER", "pegasusx-ledger"); got != "custom-ledger-topic" {
		t.Fatalf("topicFromEnv override = %q, want %q", got, "custom-ledger-topic")
	}
}

func TestRelayPublishTopics_ledgerDualWrite(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_DUAL_WRITE", "true")
	for _, evType := range []string{
		EventGLEntryPosted,
		EventGLEntryReversed,
		EventReconRunCompleted,
		EventReconBreakOpened,
		EventReconBreakResolved,
		EventAuditAnchorWritten,
		EventIAMRoleChanged,
		EventSodViolationDetected,
	} {
		payload := []byte(`{"type":"` + evType + `","entity_id":"ent-1"}`)
		if topics := RelayPublishTopics(TopicMain, payload); len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicLedger {
			t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s %s]", evType, topics, TopicMain, TopicLedger)
		}
		if topics := RelayPublishTopics(TopicLedger, payload); len(topics) != 1 || topics[0] != TopicLedger {
			t.Fatalf("RelayPublishTopics(TopicLedger, %s) = %v, want [%s]", evType, topics, TopicLedger)
		}
	}
}

func TestRelayPublishTopics_wave3aDualWrite(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_DUAL_WRITE", "true")
	for _, evType := range []string{
		EventPeriodSoftClosed, EventPeriodClosed, EventPeriodReopened,
		EventEsfSubmitted, EventEsfAccepted, EventEsfRejected, EventEsfCancelled,
	} {
		payload := []byte(`{"type":"` + evType + `","entity_id":"ent-1"}`)
		if topics := RelayPublishTopics(TopicMain, payload); len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicLedger {
			t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s %s]", evType, topics, TopicMain, TopicLedger)
		}
		if topics := RelayPublishTopics(TopicLedger, payload); len(topics) != 1 || topics[0] != TopicLedger {
			t.Fatalf("RelayPublishTopics(TopicLedger, %s) = %v, want [%s]", evType, topics, TopicLedger)
		}
	}
	for _, evType := range []string{
		EventInventoryReceived, EventInventoryIssued, EventInventoryExpiryRisk,
	} {
		payload := []byte(`{"type":"` + evType + `","entity_id":"ent-1"}`)
		if topics := RelayPublishTopics(TopicMain, payload); len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicDispatch {
			t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s %s]", evType, topics, TopicMain, TopicDispatch)
		}
		if topics := RelayPublishTopics(TopicDispatch, payload); len(topics) != 1 || topics[0] != TopicDispatch {
			t.Fatalf("RelayPublishTopics(TopicDispatch, %s) = %v, want [%s]", evType, topics, TopicDispatch)
		}
	}
	for _, evType := range []string{
		EventApprovalRequested, EventApprovalDecided,
	} {
		payload := []byte(`{"type":"` + evType + `","tenant_id":"sup-1"}`)
		if topics := RelayPublishTopics(TopicMain, payload); len(topics) != 1 || topics[0] != TopicMain {
			t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s]", evType, topics, TopicMain)
		}
	}
}

func TestRelayPublishTopics_wave3bDualWrite(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_DUAL_WRITE", "true")
	for _, evType := range []string{
		EventForecastRunCompleted, EventCreditScoreComputed,
		EventProposalCreated, EventProposalDecided, EventProposalExecuted,
		EventRoiEventRecorded,
	} {
		payload := []byte(`{"type":"` + evType + `","tenant_id":"sup-1"}`)
		if topics := RelayPublishTopics(TopicMain, payload); len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicIntel {
			t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s %s]", evType, topics, TopicMain, TopicIntel)
		}
		if topics := RelayPublishTopics(TopicIntel, payload); len(topics) != 1 || topics[0] != TopicIntel {
			t.Fatalf("RelayPublishTopics(TopicIntel, %s) = %v, want [%s]", evType, topics, TopicIntel)
		}
	}
	payloadRebalance := []byte(`{"type":"` + EventInventoryRebalanceProposed + `","tenant_id":"sup-1"}`)
	if topics := RelayPublishTopics(TopicMain, payloadRebalance); len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicDispatch {
		t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s %s]", EventInventoryRebalanceProposed, topics, TopicMain, TopicDispatch)
	}
	for _, evType := range []string{
		EventCreditLimitChanged, EventCreditHoldApplied, EventCreditHoldReleased,
	} {
		payload := []byte(`{"type":"` + evType + `","retailer_id":"ret-1"}`)
		if topics := RelayPublishTopics(TopicMain, payload); len(topics) != 1 || topics[0] != TopicMain {
			t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s]", evType, topics, TopicMain)
		}
	}
}

func TestRelayPublishTopics_wave3cDualWrite(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_DUAL_WRITE", "true")
	for _, evType := range []string{
		EventConnectorSyncCompleted, EventConnectorSyncFailed, EventConnectorShadowMismatch,
	} {
		payload := []byte(`{"type":"` + evType + `","tenant_id":"sup-1"}`)
		if topics := RelayPublishTopics(TopicMain, payload); len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicConnectors {
			t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s %s]", evType, topics, TopicMain, TopicConnectors)
		}
		if topics := RelayPublishTopics(TopicConnectors, payload); len(topics) != 1 || topics[0] != TopicConnectors {
			t.Fatalf("RelayPublishTopics(TopicConnectors, %s) = %v, want [%s]", evType, topics, TopicConnectors)
		}
	}
	for _, evType := range []string{
		EventExportPackCreated, EventAuditorAccessGranted,
		EventEntityIntercompanyPosted, EventFxRevaluationPosted, EventP2PThreeWayMatchCompleted,
	} {
		payload := []byte(`{"type":"` + evType + `","tenant_id":"sup-1"}`)
		if topics := RelayPublishTopics(TopicMain, payload); len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicLedger {
			t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s %s]", evType, topics, TopicMain, TopicLedger)
		}
		if topics := RelayPublishTopics(TopicLedger, payload); len(topics) != 1 || topics[0] != TopicLedger {
			t.Fatalf("RelayPublishTopics(TopicLedger, %s) = %v, want [%s]", evType, topics, TopicLedger)
		}
	}
}

func TestRelayPublishTopics_wave4aDualWrite(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_DUAL_WRITE", "true")
	poPayload := []byte(`{"type":"` + EventP2PPOCreated + `","tenant_id":"sup-1","po_id":"po-1"}`)
	if topics := RelayPublishTopics(TopicMain, poPayload); len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicOrders {
		t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s %s]", EventP2PPOCreated, topics, TopicMain, TopicOrders)
	}
	rcptPayload := []byte(`{"type":"` + EventP2PReceiptPosted + `","tenant_id":"sup-1","po_id":"po-1","receipt_id":"rcpt-1"}`)
	if topics := RelayPublishTopics(TopicMain, rcptPayload); len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicDispatch {
		t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s %s]", EventP2PReceiptPosted, topics, TopicMain, TopicDispatch)
	}
	pushPayload := []byte(`{"type":"` + EventConnectorPushCompleted + `","tenant_id":"sup-1","run_id":"run-1","provider":"moysklad"}`)
	if topics := RelayPublishTopics(TopicMain, pushPayload); len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicConnectors {
		t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s %s]", EventConnectorPushCompleted, topics, TopicMain, TopicConnectors)
	}
}

func TestRelayPublishTopics_wave4bDualWrite(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_DUAL_WRITE", "true")
	for _, evType := range []string{EventAnomalyDetected, EventAnomalyResolved} {
		payload := []byte(`{"type":"` + evType + `","tenant_id":"sup-1","case_id":"anom-1"}`)
		if topics := RelayPublishTopics(TopicMain, payload); len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicExceptions {
			t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s %s]", evType, topics, TopicMain, TopicExceptions)
		}
		if topics := RelayPublishTopics(TopicExceptions, payload); len(topics) != 1 || topics[0] != TopicExceptions {
			t.Fatalf("RelayPublishTopics(TopicExceptions, %s) = %v, want [%s]", evType, topics, TopicExceptions)
		}
	}
	whatIfPayload := []byte(`{"type":"` + EventWhatIfSimulated + `","tenant_id":"sup-1","simulation_id":"sim-1"}`)
	if topics := RelayPublishTopics(TopicMain, whatIfPayload); len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicIntel {
		t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s %s]", EventWhatIfSimulated, topics, TopicMain, TopicIntel)
	}
	haccpPayload := []byte(`{"type":"` + EventHaccpExcursionReported + `","tenant_id":"sup-1","status":"EXCURSION_OPEN"}`)
	if topics := RelayPublishTopics(TopicMain, haccpPayload); len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicTelemetryLogistics {
		t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s %s]", EventHaccpExcursionReported, topics, TopicMain, TopicTelemetryLogistics)
	}
	l2Payload := []byte(`{"type":"` + EventProposalL2AutoExecuted + `","tenant_id":"sup-1","proposal_id":"prop-1"}`)
	if topics := RelayPublishTopics(TopicMain, l2Payload); len(topics) != 1 || topics[0] != TopicMain {
		t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s]", EventProposalL2AutoExecuted, topics, TopicMain)
	}
	scimPayload := []byte(`{"type":"` + EventScimIdentitySynced + `","tenant_id":"sup-1","resource_id":"scim-1"}`)
	if topics := RelayPublishTopics(TopicMain, scimPayload); len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicLedger {
		t.Fatalf("RelayPublishTopics(TopicMain, %s) = %v, want [%s %s]", EventScimIdentitySynced, topics, TopicMain, TopicLedger)
	}
}

func TestRelayPublishTopics_demandSignalDualWrite(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_DUAL_WRITE", "true")
	payload := []byte(`{"type":"DEMAND_SIGNAL","retailer_id":"r1","sku":"SKU-1","day":"2026-08-02","source":"STORE_POS"}`)
	topics := RelayPublishTopics(TopicMain, payload)
	if len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicDemand {
		t.Fatalf("topics=%v want [%s %s]", topics, TopicMain, TopicDemand)
	}
}

func TestRelayPublishTopics_dualWrite(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_DUAL_WRITE", "true")
	payload := []byte(`{"type":"ORDER_CREATED","order_id":"o1"}`)
	topics := RelayPublishTopics(TopicMain, payload)
	if len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicOrders {
		t.Fatalf("topics=%v want [%s %s]", topics, TopicMain, TopicOrders)
	}
}

func TestRelayPublishTopics_disabled(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_DUAL_WRITE", "false")
	payload := []byte(`{"type":"ORDER_CREATED"}`)
	topics := RelayPublishTopics(TopicMain, payload)
	if len(topics) != 1 || topics[0] != TopicMain {
		t.Fatalf("topics=%v want [%s]", topics, TopicMain)
	}
}

func TestDualWriteDomainTopics_env(t *testing.T) {
	os.Unsetenv("KAFKA_TOPIC_DUAL_WRITE")
	if DualWriteDomainTopics() {
		t.Fatal("expected false by default")
	}
	t.Setenv("KAFKA_TOPIC_DUAL_WRITE", "true")
	if !DualWriteDomainTopics() {
		t.Fatal("expected true")
	}
}

func TestOrderConsumerTopic_cutover(t *testing.T) {
	os.Unsetenv("KAFKA_TOPIC_CONSUME_DOMAIN")
	if OrderConsumerTopic() != TopicMain {
		t.Fatalf("default topic = %q want %q", OrderConsumerTopic(), TopicMain)
	}
	t.Setenv("KAFKA_TOPIC_CONSUME_DOMAIN", "true")
	if OrderConsumerTopic() != TopicOrders {
		t.Fatalf("domain topic = %q want %q", OrderConsumerTopic(), TopicOrders)
	}
}

func TestDispatchConsumerTopic_cutover(t *testing.T) {
	os.Unsetenv("KAFKA_TOPIC_CONSUME_DOMAIN")
	if DispatchConsumerTopic() != TopicMain {
		t.Fatalf("default topic = %q want %q", DispatchConsumerTopic(), TopicMain)
	}
	t.Setenv("KAFKA_TOPIC_CONSUME_DOMAIN", "true")
	if DispatchConsumerTopic() != TopicDispatch {
		t.Fatalf("domain topic = %q want %q", DispatchConsumerTopic(), TopicDispatch)
	}
}

func TestTwinConsumerTopics_cutover(t *testing.T) {
	os.Unsetenv("KAFKA_TOPIC_CONSUME_DOMAIN")
	topics := TwinConsumerTopics()
	if len(topics) != 1 || topics[0] != TopicMain {
		t.Fatalf("default twin topics = %v", topics)
	}
	t.Setenv("KAFKA_TOPIC_CONSUME_DOMAIN", "true")
	topics = TwinConsumerTopics()
	if len(topics) < 4 {
		t.Fatalf("domain twin topics = %v", topics)
	}
	for _, tpc := range topics {
		if tpc == TopicWebhooks {
			t.Fatalf("retired TopicWebhooks must not be in twin fan-in: %v", topics)
		}
	}
}

func TestDomainTopicNeverWebhooks(t *testing.T) {
	t.Parallel()
	samples := []string{
		EventOrderCreated, EventPaymentCleared, EventDriverLocationUpdated,
		EventManifestSealed, EventDemandSignal, EventClaimFiled,
		EventGLEntryPosted, EventGLEntryReversed,
		EventReconRunCompleted, EventReconBreakOpened, EventReconBreakResolved,
		EventAuditAnchorWritten, EventIAMRoleChanged, EventSodViolationDetected,
		EventOntologyObjectChanged, EventOntologyObjectUpdated,
		EventPeriodSoftClosed, EventPeriodClosed, EventPeriodReopened,
		EventApprovalRequested, EventApprovalDecided,
		EventInventoryReceived, EventInventoryIssued, EventInventoryExpiryRisk,
		EventEsfSubmitted, EventEsfAccepted, EventEsfRejected, EventEsfCancelled,
		EventForecastRunCompleted, EventCreditScoreComputed, EventCreditLimitChanged,
		EventCreditHoldApplied, EventCreditHoldReleased,
		EventProposalCreated, EventProposalDecided, EventProposalExecuted,
		EventInventoryRebalanceProposed, EventRoiEventRecorded,
		EventConnectorSyncCompleted, EventConnectorSyncFailed, EventConnectorShadowMismatch,
		EventExportPackCreated, EventAuditorAccessGranted,
		EventEntityIntercompanyPosted, EventFxRevaluationPosted,
		EventP2PPOCreated, EventP2PReceiptPosted, EventP2PThreeWayMatchCompleted,
		EventConnectorPushCompleted,
		EventAnomalyDetected, EventAnomalyResolved,
		EventWhatIfSimulated, EventHaccpExcursionReported,
		EventProposalL2AutoExecuted, EventScimIdentitySynced,
	}
	for _, typ := range samples {
		if got := DomainTopicForEventType(typ); got == TopicWebhooks {
			t.Fatalf("%s mapped to retired TopicWebhooks", typ)
		}
	}
}
