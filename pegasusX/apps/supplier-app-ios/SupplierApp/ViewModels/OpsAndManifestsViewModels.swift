import Foundation
import MapKit
import Observation
import SwiftUI

@Observable
@MainActor
final class PromotionsViewModel {
    var promotions: [SupplierPromotion] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var showCreate = false
    var showEdit = false
    var editingPromotion: SupplierPromotion?
    var name = ""
    var discountBps = "500"
    var simResults: [String: PromoSimulateResult] = [:]
    var simulatingId: String?

    func beginEdit(_ promo: SupplierPromotion) {
        editingPromotion = promo
        name = promo.name
        discountBps = String(promo.discountBps)
        showEdit = true
        PegasusHaptics.selection()
    }

    func cancelEdit() {
        showEdit = false
        editingPromotion = nil
    }

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !promotions.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedPromotions = try await SupplierService.promotions()
            guard generation == loadGeneration else { return }
            promotions = loadedPromotions
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

    func create() async {
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, let bps = Int64(discountBps.filter(\.isNumber)), bps > 0 else { return }
        do {
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            _ = try await SupplierService.createPromotion(
                SupplierPromotionUpsertRequest(
                    name: trimmed,
                    description: "",
                    discountBps: bps,
                    scopeType: "ALL_PRODUCTS",
                    retailerScope: "ALL",
                    scopeProductId: nil
                ),
                idempotencyKey: SupplierIdempotencyKeys.promotionCreate(
                    scopeId: scope,
                    payloadFingerprint: "\(trimmed):\(bps):ALL_PRODUCTS"
                )
            )
            name = ""
            discountBps = "500"
            showCreate = false
            PegasusHaptics.confirm()
            await load(silent: true)
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !promotions.isEmpty, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func saveEdit() async {
        guard let promo = editingPromotion else { return }
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, let bps = Int64(discountBps.filter(\.isNumber)), bps > 0 else { return }
        do {
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            _ = try await SupplierService.updatePromotion(
                promotionId: promo.promotionId,
                SupplierPromotionUpsertRequest(
                    name: trimmed,
                    description: "",
                    discountBps: bps,
                    scopeType: promo.scopeType,
                    retailerScope: promo.retailerScope,
                    scopeProductId: promo.scopeProductId
                ),
                idempotencyKey: SupplierIdempotencyKeys.promotionUpdate(
                    scopeId: scope,
                    promotionId: promo.promotionId,
                    payloadFingerprint: "\(trimmed):\(bps):\(promo.scopeType)"
                )
            )
            showEdit = false
            editingPromotion = nil
            PegasusHaptics.confirm()
            await load(silent: true)
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !promotions.isEmpty, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func simulate(_ promo: SupplierPromotion) async {
        simulatingId = promo.promotionId
        defer { simulatingId = nil }
        do {
            let result = try await SupplierOperationsService.simulatePromotionPandL(
                PromoSimulateInput(
                    promotionId: promo.promotionId,
                    discountPct: Double(promo.discountBps) / 100.0,
                    expectedUnits: 500,
                    avgUnitMarginMinor: 1000
                )
            )
            simResults[promo.promotionId] = result
            PegasusHaptics.lightTap()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !promotions.isEmpty, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func deactivate(_ promotionId: String) async {
        do {
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            try await SupplierService.deactivatePromotion(
                promotionId: promotionId,
                idempotencyKey: SupplierIdempotencyKeys.promotionDeactivate(
                    scopeId: scope,
                    promotionId: promotionId
                )
            )
            PegasusHaptics.warning()
            await load(silent: true)
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !promotions.isEmpty, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class ProfileViewModel {
    var profile: SupplierProfile?
    var loading = true
    private var loadGeneration = 0
    var error: String?

    func load(silent: Bool = false, tokenStore: TokenStore) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = profile != nil
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedProfile = try await SupplierService.profile()
            guard generation == loadGeneration else { return }
            profile = loadedProfile
            if let configured = profile?.isConfigured {
                tokenStore.markConfigured(configured)
            }
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
final class EarningsViewModel {
    var earnings: SupplierEarnings?
    var loading = true
    private var loadGeneration = 0
    var error: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = earnings != nil
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedEarnings = try await SupplierService.earnings()
            guard generation == loadGeneration else { return }
            earnings = loadedEarnings
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
final class InventoryImportViewModel {
    var csvText = ""
    var fileName = ""
    var step = 0
    var sessionId: String?
    var sessionStatus = ""
    var mappings: [SupplierImportMappingCandidate] = []
    var applyResult: SupplierImportApplyResponse?
    var directResult: SupplierInventoryImportResult?
    var busy = false
    var error: String?

    func directImport() async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            let scopeId = SupplierIdempotencyKeys.supplierScopeId()
            directResult = try await SupplierService.importInventoryCSV(
                csvText,
                idempotencyKey: SupplierIdempotencyKeys.importDirect(scopeId: scopeId, csvBody: csvText)
            )
            step = 3
            PegasusHaptics.confirm()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func startWizard() async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            let scopeId = SupplierIdempotencyKeys.supplierScopeId()
            let bytes = csvText.utf8.count
            let created = try await SupplierOperationsService.createImportSession(
                fileName: fileName,
                fileSizeBytes: bytes,
                idempotencyKey: SupplierIdempotencyKeys.importCreate(scopeId: scopeId, fileName: fileName, fileSizeBytes: bytes)
            )
            let ingested = try await SupplierOperationsService.ingestImportSession(
                sessionId: created.sessionId,
                csv: csvText,
                idempotencyKey: SupplierIdempotencyKeys.importIngest(sessionId: created.sessionId, csvBody: csvText)
            )
            let mapping = try await SupplierOperationsService.getImportMapping(sessionId: created.sessionId)
            sessionId = created.sessionId
            sessionStatus = ingested.status
            mappings = mapping.mappingJson?.mappings ?? []
            step = 1
            PegasusHaptics.selection()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func approveAndApply(_ sessionId: String) async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            try await SupplierOperationsService.approveImportSession(
                sessionId: sessionId,
                idempotencyKey: SupplierIdempotencyKeys.importApprove(sessionId: sessionId)
            )
            applyResult = try await SupplierOperationsService.applyImportSession(
                sessionId: sessionId,
                idempotencyKey: SupplierIdempotencyKeys.importApply(sessionId: sessionId)
            )
            step = 3
            PegasusHaptics.confirm()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class ReturnsViewModel {
    var items: [SupplierReturnRow] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var resolvingId: String?
    var resolution = "RETURN_TO_STOCK"
    var notes = ""
    var actionLoading: String?

    func startDispute(returnId: String) {
        resolvingId = returnId
        resolution = "RETURN_TO_STOCK"
        notes = ""
        PegasusHaptics.selection()
    }

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !items.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let resp = try await SupplierOperationsService.returns(status: "PENDING", limit: 100, offset: 0)
            guard generation == loadGeneration else { return }
            items = resp.data
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

    func resolve(returnId: String) async {
        actionLoading = returnId
        defer { actionLoading = nil }
        do {
            let key = SupplierIdempotency.resolveReturn(returnId: returnId, resolution: resolution)
            try await SupplierOperationsService.resolveReturn(
                returnId: returnId,
                resolution: resolution,
                notes: notes,
                idempotencyKey: key
            )
            resolvingId = nil
            notes = ""
            PegasusHaptics.confirm()
            await load()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !items.isEmpty, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class ManifestDetailViewModel {
    var detail: SupplierManifestDetail?
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var actionError: String?
    var busy = false
    var injectOrderId = ""

    func handleReconnect() {
        if busy {
            busy = false
            actionError = ManifestDetailView.connectionRestoredNotice()
        }
    }

    private func idem(_ manifestId: String, prefix: String, extra: String = "") -> String {
        let trimmed = extra.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty {
            return "\(prefix):\(manifestId)"
        }
        return "\(prefix):\(manifestId):\(trimmed)"
    }

    func load(manifestId: String, silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = detail != nil
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedDetail = try await SupplierOperationsService.manifestDetail(manifestId)
            guard generation == loadGeneration else { return }
            detail = loadedDetail
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

    func startLoading(manifestId: String) async {
        await runAction(manifestId: manifestId) {
            try await SupplierOperationsService.startManifestLoading(manifestId, idempotencyKey: self.idem(manifestId, prefix: "start-loading"))
        }
    }

    func injectOrder(manifestId: String) async {
        let orderId = injectOrderId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !orderId.isEmpty else { return }
        await runAction(manifestId: manifestId) {
            try await SupplierOperationsService.injectManifestOrder(
                manifestId,
                request: SupplierManifestInjectOrderRequest(orderId: orderId, volumeVu: nil),
                idempotencyKey: self.idem(manifestId, prefix: "inject-order", extra: orderId)
            )
            self.injectOrderId = ""
        }
    }

    func seal(manifestId: String) async {
        await runAction(manifestId: manifestId) {
            try await SupplierOperationsService.sealManifest(manifestId, idempotencyKey: self.idem(manifestId, prefix: "seal"))
        }
    }

    private func runAction(manifestId: String, _ action: () async throws -> Void) async {
        busy = true
        actionError = nil
        defer { busy = false }
        do {
            try await action()
            PegasusHaptics.confirm()
            await load(manifestId: manifestId)
        } catch {
            actionError = error.localizedDescription
            SupplierActionFailure.record(underlying: error, hasContent: detail != nil, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class ManifestExceptionsViewModel {
    var rows: [SupplierManifestExceptionRow] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var escalatedOnly = false

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !rows.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedRows = try await SupplierOperationsService.manifestExceptions(escalatedOnly: escalatedOnly)
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
final class ManifestsViewModel {
    var rows: [SupplierManifestRow] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !rows.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedRows = try await SupplierOperationsService.manifests()
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
final class FleetLiveMapViewModel {
    var routes: [SupplierFleetLiveRoute] = []
    var exceptionCells: [ExceptionMapCell] = []
    var zoneOverrides: [ControlTowerZoneOverride] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var overlayError: String?
    var cameraPosition: MapCameraPosition = .region(
        MKCoordinateRegion(
            center: CLLocationCoordinate2D(latitude: packMapCoordinate().lat, longitude: packMapCoordinate().lng),
            span: MKCoordinateSpan(latitudeDelta: 0.18, longitudeDelta: 0.18)
        )
    )
    var animator = FleetDriverMarkerAnimator()
    var hasFittedInitialCamera = false
    var publishAction = "REROUTE"
    var publishStatus: String?
    var publishing = false
    var showPublishSheet = false
    var mapRegion = MKCoordinateRegion(
        center: CLLocationCoordinate2D(latitude: packMapCoordinate().lat, longitude: packMapCoordinate().lng),
        span: MKCoordinateSpan(latitudeDelta: 0.18, longitudeDelta: 0.18)
    )

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !routes.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            async let fleetTask = SupplierOperationsService.fleetLiveMap()
            async let exceptionsTask = SupplierOperationsService.exceptionMap()
            async let overridesTask = SupplierOperationsService.controlTowerZoneOverrides()

            let response = try await fleetTask
            let cellsOutcome: PulseHonesty.Result<ExceptionMapCell>
            do {
                let map = try await exceptionsTask
                cellsOutcome = PulseHonesty.apply(
                    ok: true,
                    incoming: map.cells,
                    previous: exceptionCells,
                    failure: "exception_map_failed"
                )
            } catch {
                cellsOutcome = PulseHonesty.apply(
                    ok: false,
                    incoming: nil,
                    previous: exceptionCells,
                    failure: error.localizedDescription
                )
            }
            let overridesOutcome: PulseHonesty.Result<ControlTowerZoneOverride>
            do {
                let loadedOverrides = try await overridesTask
                overridesOutcome = PulseHonesty.apply(
                    ok: true,
                    incoming: loadedOverrides,
                    previous: zoneOverrides,
                    failure: "zone_overrides_failed"
                )
            } catch {
                overridesOutcome = PulseHonesty.apply(
                    ok: false,
                    incoming: nil,
                    previous: zoneOverrides,
                    failure: error.localizedDescription
                )
            }
            guard generation == loadGeneration else { return }
            routes = response.routes
            animator.updateTargets(routes)
            exceptionCells = cellsOutcome.events
            zoneOverrides = overridesOutcome.events
            overlayError = cellsOutcome.error ?? overridesOutcome.error
            if !silent || !hasFittedInitialCamera {
                fitCamera()
                if !routes.isEmpty {
                    hasFittedInitialCamera = true
                }
            }
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

    private func fitCamera() {
        var coordinates: [CLLocationCoordinate2D] = []
        for route in routes {
            if let geometry = route.routeGeometry {
                coordinates.append(contentsOf: geometry.coordinates.map {
                    CLLocationCoordinate2D(latitude: $0.lat, longitude: $0.lng)
                })
            }
            if let location = route.driverLocation, route.liveLocationAvailable {
                coordinates.append(CLLocationCoordinate2D(latitude: location.latitude, longitude: location.longitude))
            }
        }
        guard !coordinates.isEmpty else { return }
        let lats = coordinates.map(\.latitude)
        let lngs = coordinates.map(\.longitude)
        guard let minLat = lats.min(), let maxLat = lats.max(), let minLng = lngs.min(), let maxLng = lngs.max() else {
            return
        }
        let center = CLLocationCoordinate2D(
            latitude: (minLat + maxLat) / 2,
            longitude: (minLng + maxLng) / 2
        )
        let span = MKCoordinateSpan(
            latitudeDelta: max(0.03, (maxLat - minLat) * 1.4),
            longitudeDelta: max(0.03, (maxLng - minLng) * 1.4)
        )
        cameraPosition = .region(MKCoordinateRegion(center: center, span: span))
    }

    func startPolling() async {
        while !Task.isCancelled {
            try? await Task.sleep(nanoseconds: 15_000_000_000)
            await load(silent: true)
        }
    }

    func publishZoneOverride() async {
        publishing = true
        publishStatus = nil
        defer { publishing = false }
        let polygon = polygonFromCurrentCamera()
        let fingerprint = polygon.coordinates.description
        let scope = SupplierIdempotencyKeys.supplierScopeId()
        let key = SupplierIdempotencyKeys.controlTowerZoneOverride(
            scopeId: scope,
            action: publishAction,
            polygonFingerprint: fingerprint
        )
        let body = ControlTowerZoneOverrideCreateRequest(
            action: publishAction,
            ttlSeconds: 1800,
            polygonGeojson: polygon
        )
        do {
            let row = try await SupplierOperationsService.createControlTowerZoneOverride(body, idempotencyKey: key)
            publishStatus = L10n.format("mobile_supplier.chrome.override_active", "\(row.overrideId.prefix(8))")
            PegasusHaptics.confirm()
            do {
                zoneOverrides = try await SupplierOperationsService.controlTowerZoneOverrides()
                overlayError = nil
            } catch {
                if !zoneOverrides.contains(where: { $0.overrideId == row.overrideId }) {
                    zoneOverrides.insert(row, at: 0)
                }
                overlayError = error.localizedDescription
            }
        } catch {
            publishStatus = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    private func polygonFromCurrentCamera() -> GeoJSONPolygonPayload {
        let center = mapRegion.center
        let latD = mapRegion.span.latitudeDelta / 2
        let lngD = mapRegion.span.longitudeDelta / 2
        let ring: [[Double]] = [
            [center.longitude - lngD, center.latitude - latD],
            [center.longitude + lngD, center.latitude - latD],
            [center.longitude + lngD, center.latitude + latD],
            [center.longitude - lngD, center.latitude + latD],
            [center.longitude - lngD, center.latitude - latD],
        ]
        return GeoJSONPolygonPayload(coordinates: [ring])
    }
}

@Observable
@MainActor
final class FleetOrdersViewModel {
    var rows: [SupplierFleetOrderRow] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !rows.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedRows = try await SupplierOperationsService.fleetOrders()
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
