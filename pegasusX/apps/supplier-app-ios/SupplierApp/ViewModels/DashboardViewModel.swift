import Foundation
import Observation

private struct DashboardDemandPanel: Sendable {
    let generatedAt: String?
    let confidence: ForecastConfidence
}

@Observable
@MainActor
final class DashboardViewModel {
    var dashboard: SupplierDashboard?
    var meiSummary: SupplierMEIONetworkSummary?
    var meiError: String?
    var pulseEvents: [SupplierPulseEvent] = []
    var pulseLoading = true
    var pulseError: String?
    var demandConfidence: ForecastConfidence?
    var demandGeneratedAt: String?
    var demandError: String?
    var loading = true
    var error: String?
    var pack: MarketPack?
    var commandJump: CommandStatusJump?

    private var loadGeneration = 0

    func loadPack(token: String?) async {
        guard let token, !token.isEmpty else { return }
        pack = await MarketPackBinder.fetch(baseUrl: APIClient.shared.cellBootstrap, token: token)?.pack
    }

    func runDashboardLoop(tokenStore: TokenStore, intervalNs: UInt64 = 60_000_000_000) async {
        await loadPack(token: tokenStore.token)
        await load(tokenStore: tokenStore)
        while !Task.isCancelled {
            try? await Task.sleep(nanoseconds: intervalNs)
            await load(tokenStore: tokenStore, silent: true)
        }
    }

    func load(tokenStore: TokenStore? = nil, silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = dashboard != nil
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedDashboard = try await SupplierService.dashboard()
            guard generation == loadGeneration else { return }
            dashboard = loadedDashboard
            let meiOutcome = await loadMEI(previous: meiSummary)
            guard generation == loadGeneration else { return }
            meiSummary = meiOutcome.value
            meiError = meiOutcome.error
            let demandOutcome = await loadDemand(
                previous: demandConfidence.map {
                    DashboardDemandPanel(generatedAt: demandGeneratedAt, confidence: $0)
                }
            )
            guard generation == loadGeneration else { return }
            demandGeneratedAt = demandOutcome.value?.generatedAt
            demandConfidence = demandOutcome.value?.confidence
            demandError = demandOutcome.error
            pulseLoading = true
            pulseError = nil
            do {
                let events = try await SupplierOperationsService.pulse().events
                guard generation == loadGeneration else { return }
                let result = PulseHonesty.apply(ok: true, incoming: events, previous: pulseEvents)
                pulseEvents = result.events
                pulseError = result.error
            } catch {
                guard generation == loadGeneration else { return }
                let result = PulseHonesty.apply(ok: false, incoming: nil, previous: pulseEvents)
                pulseEvents = result.events
                pulseError = result.error
            }
            pulseLoading = false
            if let configured = dashboard?.isConfigured {
                tokenStore?.markConfigured(configured)
            }
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            if !silent {
                dashboard = nil
            }
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

    private func loadMEI(previous: SupplierMEIONetworkSummary?) async -> PulseHonesty.ObjectResult<SupplierMEIONetworkSummary> {
        do {
            let loaded = try await SupplierOperationsService.meiNetworkSummary()
            return PulseHonesty.applyObject(ok: true, incoming: loaded, previous: previous, failure: "meio_failed")
        } catch {
            return PulseHonesty.applyObject(
                ok: false,
                incoming: nil,
                previous: previous,
                failure: error.localizedDescription
            )
        }
    }

    private func loadDemand(previous: DashboardDemandPanel?) async -> PulseHonesty.ObjectResult<DashboardDemandPanel> {
        do {
            let loaded = try await SupplierOperationsService.demandToday()
            let panel = DashboardDemandPanel(
                generatedAt: loaded.generatedAt,
                confidence: ForecastConfidenceSupport.fromDemand(loaded)
            )
            return PulseHonesty.applyObject(ok: true, incoming: panel, previous: previous, failure: "demand_failed")
        } catch {
            return PulseHonesty.applyObject(
                ok: false,
                incoming: nil,
                previous: previous,
                failure: error.localizedDescription
            )
        }
    }
}
