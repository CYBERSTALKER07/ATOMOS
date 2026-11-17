import Foundation
import Observation

@Observable
@MainActor
final class KnowledgeGraphViewModel {
    var graph: SupplierKnowledgeGraph?
    var loading = true
    private var loadGeneration = 0
    var error: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = graph != nil
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedGraph = try await SupplierOperationsService.knowledgeGraph()
            guard generation == loadGeneration else { return }
            graph = loadedGraph
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }
}

private struct DemandPanel {
    let generatedAt: String?
    let confidence: ForecastConfidence
}

@Observable
@MainActor
final class PlanningBrainViewModel {
    var tab: PlanBrainTab = .planning
    var sandop: PlanningSAndOPSnapshot?
    var scenario: PlanningScenarioResult?
    var downtimeHours = 8.0
    var demandDeltaPct = 10.0
    var loading = true
    private var loadGeneration = 0
    var running = false
    var error: String?
    var demandConfidence: ForecastConfidence?
    var demandGeneratedAt: String?
    var demandError: String?
    var retailerId = ""
    var sparsity: SparsityGateResult?
    var sparsityError: String?
    var pushStatus: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = sandop != nil
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedSandop = try await SupplierOperationsService.planningSAndOP()
            guard generation == loadGeneration else { return }
            sandop = loadedSandop
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
        guard generation == loadGeneration else { return }
        let previousDemand = demandConfidence.map {
            DemandPanel(generatedAt: demandGeneratedAt, confidence: $0)
        }
        let demandOutcome = await loadDemand(previous: previousDemand)
        guard generation == loadGeneration else { return }
        demandGeneratedAt = demandOutcome.value?.generatedAt
        demandConfidence = demandOutcome.value?.confidence
        demandError = demandOutcome.error
    }

    private func loadDemand(previous: DemandPanel?) async -> PulseHonesty.ObjectResult<DemandPanel> {
        do {
            let loaded = try await SupplierOperationsService.demandToday()
            return PulseHonesty.applyObject(
                ok: true,
                incoming: DemandPanel(
                    generatedAt: loaded.generatedAt,
                    confidence: ForecastConfidenceSupport.fromDemand(loaded)
                ),
                previous: previous,
                failure: "demand_failed"
            )
        } catch {
            return PulseHonesty.applyObject(
                ok: false,
                incoming: nil,
                previous: previous,
                failure: error.localizedDescription
            )
        }
    }

    func runScenario() async {
        running = true
        error = nil
        defer { running = false }
        do {
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            scenario = try await SupplierOperationsService.runPlanningScenario(
                factoryDowntimeHours: Int(downtimeHours),
                demandDeltaPct: demandDeltaPct,
                idempotencyKey: SupplierIdempotencyKeys.planningScenario(
                    scopeId: scope,
                    factoryDowntimeHours: Int(downtimeHours),
                    demandDeltaPct: demandDeltaPct
                )
            )
            PegasusHaptics.confirm()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: sandop != nil, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func checkSparsity() async {
        let id = retailerId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !id.isEmpty else { return }
        do {
            let loaded = try await SupplierOperationsService.planningSparsity(retailerId: id)
            let outcome = PulseHonesty.applyObject(ok: true, incoming: loaded, previous: sparsity, failure: "sparsity_failed")
            sparsity = outcome.value
            sparsityError = outcome.error
            PegasusHaptics.lightTap()
        } catch {
            let outcome = PulseHonesty.applyObject(
                ok: false,
                incoming: nil as SparsityGateResult?,
                previous: sparsity,
                failure: error.localizedDescription
            )
            sparsity = outcome.value
            sparsityError = outcome.error
            PegasusHaptics.error()
        }
    }

    func runPush() async {
        do {
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            let resp = try await SupplierOperationsService.planningPredictivePush(
                idempotencyKey: SupplierIdempotencyKeys.planningPredictivePush(scopeId: scope)
            )
            pushStatus = L10n.format("mobile_supplier.chrome.preview_transfers", resp.source, "\(resp.transfers)")
            PegasusHaptics.confirm()
        } catch {
            let text = error.localizedDescription
            if text.contains("factory_planning_disabled") || text.contains("HTTP 409") {
                pushStatus = L10n.format("mobile_supplier.chrome.factory_planning_off")
            } else {
                pushStatus = text
            }
            SupplierActionFailure.record(underlying: error, hasContent: sandop != nil, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

private struct NetworkModeRead {
    let mode: String
    let planningEnabled: Bool
}

@Observable
@MainActor
final class PlanningSettingsViewModel {
    var data: SeasonalTemplatesResponse?
    var loading = true
    private var loadGeneration = 0
    var saving = false
    var error: String?
    var formError: String?
    var templateId = ""
    var name = ""
    var startDate = ""
    var endDate = ""
    var multiplier = ""
    var networkMode = ""
    var planningEnabled = false
    var modeKnown = false
    var modeError: String?
    var opsStatus: String?
    var killReason = ""
    var opsBusy = false

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = data != nil
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedData = try await SupplierOperationsService.seasonalOverrides()
            guard generation == loadGeneration else { return }
            data = loadedData
            let previousMode: NetworkModeRead? = modeKnown
                ? NetworkModeRead(mode: networkMode, planningEnabled: planningEnabled)
                : nil
            let modeOutcome = await loadNetworkMode(previous: previousMode)
            guard generation == loadGeneration else { return }
            if let value = modeOutcome.value {
                networkMode = value.mode
                planningEnabled = value.planningEnabled
                modeKnown = true
            }
            modeError = modeOutcome.error
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }

    private func loadNetworkMode(previous: NetworkModeRead?) async -> PulseHonesty.ObjectResult<NetworkModeRead> {
        do {
            let loaded = try await SupplierOperationsService.networkMode()
            return PulseHonesty.applyObject(
                ok: true,
                incoming: NetworkModeRead(mode: loaded.mode, planningEnabled: loaded.planningEnabled),
                previous: previous,
                failure: "network_mode_failed"
            )
        } catch {
            return PulseHonesty.applyObject(
                ok: false,
                incoming: nil,
                previous: previous,
                failure: error.localizedDescription
            )
        }
    }

    func createOverride() async {
        guard !startDate.isEmpty, !endDate.isEmpty else {
            formError = L10n.format("mobile_supplier.chrome.start_end_dates_required")
            return
        }
        saving = true
        formError = nil
        defer { saving = false }
        do {
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            let multVal = Double(multiplier.trimmingCharacters(in: .whitespacesAndNewlines))
            let row = try await SupplierOperationsService.createSeasonalOverride(
                SeasonalOverrideInput(
                    templateId: templateId.isEmpty ? nil : templateId,
                    startDate: startDate,
                    endDate: endDate,
                    name: name.isEmpty ? nil : name,
                    multiplier: multVal
                ),
                idempotencyKey: SupplierIdempotencyKeys.seasonalOverrideCreate(
                    scopeId: scope,
                    startDate: startDate,
                    endDate: endDate
                )
            )
            var overrides = data?.overrides ?? []
            overrides.insert(row, at: 0)
            data = SeasonalTemplatesResponse(
                builtinTemplates: data?.builtinTemplates ?? [],
                overrides: overrides
            )
            name = ""
            startDate = ""
            endDate = ""
            templateId = ""
            multiplier = ""
            PegasusHaptics.confirm()
        } catch {
            formError = error.localizedDescription
            SupplierActionFailure.record(underlying: error, hasContent: data != nil, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func putMode(_ mode: String) async {
        opsBusy = true
        defer { opsBusy = false }
        do {
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            let resp = try await SupplierOperationsService.putNetworkMode(
                NetworkModeUpdateRequest(mode: mode, reason: nil),
                idempotencyKey: SupplierIdempotencyKeys.networkModePut(scopeId: scope, mode: mode)
            )
            networkMode = resp.newMode
            opsStatus = L10n.format("mobile_supplier.chrome.mode_changed", resp.oldMode, resp.newMode)
            PegasusHaptics.selection()
        } catch {
            opsStatus = error.localizedDescription
            SupplierActionFailure.record(underlying: error, hasContent: data != nil, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func runPullMatrix() async {
        opsBusy = true
        defer { opsBusy = false }
        do {
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            let resp = try await SupplierOperationsService.planningPullMatrix(
                idempotencyKey: SupplierIdempotencyKeys.planningPullMatrix(scopeId: scope)
            )
            opsStatus = L10n.format("mobile_supplier.chrome.pull_matrix_status", resp.status, "\(resp.transfers)")
            PegasusHaptics.confirm()
        } catch {
            let text = error.localizedDescription
            if text.contains("factory_planning_disabled") || text.contains("HTTP 409") {
                opsStatus = L10n.format("mobile_supplier.chrome.factory_planning_off")
            } else {
                opsStatus = text
            }
            SupplierActionFailure.record(underlying: error, hasContent: data != nil, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func runPredictivePush() async {
        opsBusy = true
        defer { opsBusy = false }
        do {
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            let resp = try await SupplierOperationsService.planningPredictivePush(
                idempotencyKey: SupplierIdempotencyKeys.planningPredictivePush(scopeId: scope)
            )
            opsStatus = L10n.format("mobile_supplier.chrome.predictive_push_status", resp.source, "\(resp.transfers)", "\(resp.skus)", resp.grain.isEmpty ? L10n.format("mobile_supplier.chrome.baseline_grain") : resp.grain)
            PegasusHaptics.confirm()
        } catch {
            let text = error.localizedDescription
            if text.contains("factory_planning_disabled") || text.contains("HTTP 409") {
                opsStatus = L10n.format("mobile_supplier.chrome.factory_planning_off")
            } else {
                opsStatus = text
            }
            SupplierActionFailure.record(underlying: error, hasContent: data != nil, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func runKillSwitch() async {
        if let validationError = PlanningSettingsView.killSwitchReasonValidation(reason: killReason) {
            opsStatus = validationError
            PegasusHaptics.warning()
            return
        }
        let reason = killReason.trimmingCharacters(in: .whitespacesAndNewlines)
        opsBusy = true
        defer { opsBusy = false }
        do {
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            let resp = try await SupplierOperationsService.planningKillSwitch(
                KillSwitchRequest(reason: reason),
                idempotencyKey: SupplierIdempotencyKeys.planningKillSwitch(scopeId: scope, reason: reason)
            )
            opsStatus = L10n.format("mobile_supplier.chrome.kill_switch_status", "\(resp.cancelledTransfers)", resp.mode)
            killReason = ""
            PegasusHaptics.warning()
        } catch {
            opsStatus = error.localizedDescription
            SupplierActionFailure.record(underlying: error, hasContent: data != nil, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class ReplenishmentPoliciesViewModel {
    var policy: SupplierReplenishmentPolicy?
    var loading = true
    private var loadGeneration = 0
    var error: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = policy != nil
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedPolicy = try await SupplierOperationsService.replenishmentPolicies()
            guard generation == loadGeneration else { return }
            policy = loadedPolicy
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }
}

@Observable
@MainActor
final class ClaimsViewModel {
    var claims: [SupplierClaim] = []
    var statusFilter = "OPEN"
    var settlementMode = "LEDGER_ONLY"
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var busyId: String?
    var lastSettlement: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !claims.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedClaims = try await SupplierOperationsService.listSupplierClaims(
                status: statusFilter.isEmpty ? nil : statusFilter
            )
            guard generation == loadGeneration else { return }
            claims = loadedClaims
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }

    func approve(_ claimId: String) async {
        busyId = claimId
        lastSettlement = nil
        defer { busyId = nil }
        do {
            let resp = try await SupplierOperationsService.approveClaim(
                claimId: claimId,
                request: ApproveClaimRequest(
                    resolutionNote: "approved_via_supplier_ios",
                    settlementMode: settlementMode,
                    skipGatewayRefund: settlementMode != "GATEWAY_REFUND"
                ),
                idempotencyKey: SupplierIdempotencyKeys.claimApprove(
                    claimId: claimId,
                    settlementMode: settlementMode
                )
            )
            if let s = resp.settlement {
                let formattedAmount = s.amountMinor.map { formatMinorUnits($0, fractionDigits: 2, trimTrailingZeros: false) } ?? MissingNumber.label(nil as Int64?)
                lastSettlement = L10n.format(
                    "mobile_supplier.chrome.settlement_summary",
                    s.mode,
                    formattedAmount,
                    s.gatewayRefunded ? "true" : "false",
                    s.chargebackId ?? "—"
                )
            }
            PegasusHaptics.confirm()
            await load()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !claims.isEmpty, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func reject(_ claimId: String) async {
        busyId = claimId
        defer { busyId = nil }
        do {
            _ = try await SupplierOperationsService.rejectClaim(
                claimId: claimId,
                request: RejectClaimRequest(resolutionNote: "rejected_via_supplier_ios"),
                idempotencyKey: SupplierIdempotencyKeys.claimReject(claimId: claimId)
            )
            PegasusHaptics.warning()
            await load()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !claims.isEmpty, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class EarlyCompleteViewModel {
    var driverId = ""
    var busy = false
    var error: String?
    var success: String?

    func handleReconnect() {
        if busy {
            busy = false
            error = L10n.format("mobile_supplier.chrome.connection_restored_approval")
        }
    }

    func approve() async {
        let trimmed = driverId.trimmingCharacters(in: .whitespaces)
        guard !trimmed.isEmpty else {
            error = L10n.format("mobile_supplier.chrome.driver_id_required")
            return
        }
        busy = true
        error = nil
        success = nil
        defer { busy = false }
        do {
            try await SupplierOperationsService.approveEarlyComplete(driverId: trimmed)
            success = L10n.format("mobile_supplier.chrome.early_complete_approved", "\(trimmed.prefix(12))")
            driverId = ""
            PegasusHaptics.confirm()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class ExceptionsViewModel {
    var rows: [SupplierExceptionRow] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var busyKey: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !rows.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedRows = try await SupplierOperationsService.exceptions()
            guard generation == loadGeneration else { return }
            rows = loadedRows
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }

    func resolve(_ row: SupplierExceptionRow) async {
        let kind = row.kind.uppercased()
        let key = "\(kind):\(row.orderId)"
        busyKey = key
        defer { busyKey = nil }
        do {
            let resolveId: String
            let creditNoteId: String?
            if kind == "CREDIT_NOTE_DRAFT" {
                creditNoteId = row.note.flatMap { $0.isEmpty ? nil : $0 }
                resolveId = creditNoteId ?? row.orderId
            } else {
                creditNoteId = nil
                resolveId = row.orderId
            }
            try await SupplierOperationsService.resolveException(
                kind: kind,
                id: resolveId,
                creditNoteId: creditNoteId
            )
            PegasusHaptics.confirm()
            await load()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !rows.isEmpty, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class ShopClosedViewModel {
    var rows: [ShopClosedAttemptRow] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var busyId: String?
    var statusMessage: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !rows.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedRows = try await SupplierOperationsService.shopClosedActive()
            guard generation == loadGeneration else { return }
            rows = loadedRows
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }

    func handleReconnect() async {
        if busyId != nil {
            busyId = nil
            statusMessage = L10n.format("mobile_supplier.chrome.connection_restored_resolution")
        }
        await load()
    }

    func resolve(_ attemptId: String, action: String) async {
        busyId = attemptId
        statusMessage = nil
        defer { busyId = nil }
        do {
            _ = try await SupplierOperationsService.resolveShopClosed(
                ShopClosedResolveRequest(attemptId: attemptId, action: action),
                idempotencyKey: "shop-closed-resolve:\(attemptId):\(action)"
            )
            statusMessage = L10n.format("mobile_supplier.chrome.resolved_status", action)
            PegasusHaptics.confirm()
            await load()
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class ActivityViewModel {
    var rows: [SupplierActivityEvent] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !rows.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedRows = try await SupplierOperationsService.activity()
            guard generation == loadGeneration else { return }
            rows = loadedRows
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }
}

@Observable
@MainActor
final class AIRecommendationsViewModel {
    var filter = "PENDING"
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var items: [SupplierAIRecommendation] = []
    var pendingDecisionId: String?
    var feedback: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !items.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let resp = try await SupplierOperationsService.aiRecommendations(status: filter, limit: 50)
            guard generation == loadGeneration else { return }
            items = resp.items
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }

    func record(_ rec: SupplierAIRecommendation, decision: String) async {
        pendingDecisionId = rec.recommendationId
        feedback = nil
        defer { pendingDecisionId = nil }
        do {
            let request = SupplierAIRecommendationDecisionRequest(
                recommendationId: rec.recommendationId,
                decision: decision,
                note: nil
            )
            let key = "ai-rec-\(rec.recommendationId)-\(decision)"
            let resp = try await SupplierOperationsService.recordAiRecommendationDecision(request, idempotencyKey: key)
            if let idx = items.firstIndex(where: { $0.recommendationId == resp.recommendation.recommendationId }) {
                items[idx] = resp.recommendation
            }
            feedback = L10n.format("mobile_supplier.chrome.decision_recorded", decision.lowercased(), rec.aggregateType, rec.aggregateId)
            PegasusHaptics.confirm()
        } catch {
            feedback = L10n.format("mobile_supplier.chrome.decision_failed", error.localizedDescription)
            PegasusHaptics.error()
        }
    }
}
