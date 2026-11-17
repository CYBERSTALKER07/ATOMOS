//
//  CashierViewModel.swift
//  WarehouseAppIOS
//
//

import Foundation
import SwiftUI
import Observation

struct DenominationRowItem: Identifiable, Sendable {
    var id: Int64 { noteSom }
    let noteSom: Int64
    var count: Int64

    var totalSom: Int64 {
        noteSom * count
    }

    var totalMinor: Int64 {
        noteSom * count * 100
    }
}

struct CashierDriverOption: Identifiable, Hashable, Sendable {
    var id: String { driverId }
    let driverId: String
    let label: String
}

struct CashierRouteOption: Identifiable, Hashable, Sendable {
    var id: String { routeId }
    let routeId: String
    let label: String
    let linkedDriverId: String?
    let driverNameHint: String?

    init(
        routeId: String,
        label: String,
        linkedDriverId: String? = nil,
        driverNameHint: String? = nil
    ) {
        self.routeId = routeId
        self.label = label
        self.linkedDriverId = linkedDriverId
        self.driverNameHint = driverNameHint
    }
}

@Observable
@MainActor
final class CashierViewModel {
    private static let isoFormatter = ISO8601DateFormatter()

    static let supportedDenominations: [Int64] = [
        200000,
        100000,
        50000,
        20000,
        10000,
        5000,
        2000,
        1000,
    ]

    var cellId: String
    var depotId: String
    var supplierId: String
    var driverId: String = ""
    var routeId: String = ""
    var bagSealBarcode: String = ""

    var driverOptions: [CashierDriverOption] = []
    var routeOptions: [CashierRouteOption] = []
    var isLoadingSelectors: Bool = false
    var selectorsLoaded: Bool = false
    var manualEntryMode: Bool = false

    var session: CashierSessionView? = nil
    var attemptNumber: Int64 = 1

    var denominations: [Int64: Int64] = [
        200000: 0,
        100000: 0,
        50000: 0,
        20000: 0,
        10000: 0,
        5000: 0,
        2000: 0,
        1000: 0,
    ]

    var countedCashMinor: Int64 {
        denominations.reduce(0) { total, pair in
            total + (pair.key * pair.value * 100)
        }
    }

    var countedCashSom: Int64 {
        countedCashMinor / 100
    }

    var denominationRows: [DenominationRowItem] {
        Self.supportedDenominations.map { note in
            DenominationRowItem(noteSom: note, count: denominations[note] ?? 0)
        }
    }

    var isBlindCountMasked: Bool = true
    var isRecountRequired: Bool = false
    var isBalanced: Bool = false
    var isQuarantined: Bool = false
    var clearanceTokenHash: String? = nil
    var microSkimmingFlagged: Bool = false

    var isLoading: Bool = false
    var statusMessage: String? = nil
    var errorMessage: String? = nil

    var amlExposureMinor: Int64 = 0
    var amlCeilingMinor: Int64 = 0
    var amlExceeded: Bool = false
    var lastVaultEntry: DepotVaultLedgerEntry? = nil

    // Sheet & Form State (Consolidates UI state out of CashierBlindCountView)
    var showRRNSheet: Bool = false
    var showCITSheet: Bool = false
    var showAMLSheet: Bool = false

    var rrnOrderId: String = ""
    var rrnGateway: String = ""
    var rrnDisputedText: String = ""
    var rrnTerminalId: String = ""
    var rrnCardLastFour: String = ""
    var rrnAuthCode: String = ""

    var citBadgeId: String = ""
    var citSealBarcode: String = ""
    var citHandoverText: String = ""

    var amlTin: String = ""
    var amlExposureText: String = ""

    var totalBanknotesCount: Int64 {
        denominations.values.reduce(0, +)
    }

    private let startSessionHandler: @Sendable (StartCashierSessionRequest) async throws -> CashierSessionView
    private let submitBlindCountHandler: @Sendable (String, BlindCountRequest) async throws -> CashierSessionView
    private let reclassifyTenderHandler: @Sendable (RRNTenderReclassificationRequest) async throws -> CashierSessionView
    private let executeCITHandoverHandler: @Sendable (CITHandoverRequest) async throws -> DepotVaultLedgerEntry
    private let checkAMLHandler: @Sendable (AMLCheckRequest) async throws -> AMLCheckResponse
    private let loadDriversHandler: @Sendable () async throws -> [CashierDriverOption]
    private let loadManifestsHandler: @Sendable () async throws -> [CashierRouteOption]
    private let loadFleetLiveMapHandler: @Sendable (String?) async throws -> ([CashierDriverOption], [CashierRouteOption])

    init(
        initialCellId: String? = nil,
        initialDepotId: String? = nil,
        initialSupplierId: String? = nil,
        startSessionHandler: @escaping @Sendable (StartCashierSessionRequest) async throws -> CashierSessionView = { req in
            try await WarehouseService.startCashierSession(req: req)
        },
        submitBlindCountHandler: @escaping @Sendable (String, BlindCountRequest) async throws -> CashierSessionView = { sessionId, req in
            try await WarehouseService.submitBlindCount(sessionId: sessionId, req: req)
        },
        reclassifyTenderHandler: @escaping @Sendable (RRNTenderReclassificationRequest) async throws -> CashierSessionView = { req in
            try await WarehouseService.reclassifyTender(req: req)
        },
        executeCITHandoverHandler: @escaping @Sendable (CITHandoverRequest) async throws -> DepotVaultLedgerEntry = { req in
            try await WarehouseService.executeCITHandover(req: req)
        },
        checkAMLHandler: @escaping @Sendable (AMLCheckRequest) async throws -> AMLCheckResponse = { req in
            try await WarehouseService.checkAML(req: req)
        },
        loadDriversHandler: @escaping @Sendable () async throws -> [CashierDriverOption] = {
            let response = try await WarehouseService.drivers()
            return response.drivers.map { driver in
                let baseLabel = driver.name.isEmpty ? driver.driverId : "\(driver.name) (\(driver.driverId))"
                let label = driver.isActive ? baseLabel : "\(baseLabel) · Inactive"
                return CashierDriverOption(driverId: driver.driverId, label: label)
            }
        },
        loadManifestsHandler: @escaping @Sendable () async throws -> [CashierRouteOption] = {
            let response = try await WarehouseService.manifests()
            return response.manifests.map { manifest in
                let detail = [manifest.driverName, manifest.vehicleLabel]
                    .filter { !$0.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }
                    .joined(separator: " · ")
                let label = detail.isEmpty ? manifest.manifestId : "\(manifest.manifestId) · \(detail)"
                return CashierRouteOption(
                    routeId: manifest.manifestId,
                    label: label,
                    linkedDriverId: nil,
                    driverNameHint: manifest.driverName
                )
            }
        },
        loadFleetLiveMapHandler: @escaping @Sendable (String?) async throws -> ([CashierDriverOption], [CashierRouteOption]) = { warehouseId in
            let response = try await WarehouseService.fleetLiveMap(warehouseId: warehouseId)
            var drivers: [CashierDriverOption] = []
            var routes: [CashierRouteOption] = []
            for route in response.routes {
                let effectiveRouteId = route.routeId.isEmpty ? route.manifestId : route.routeId
                let driverDisplay = (route.driverName?.isEmpty == false ? route.driverName : nil) ?? route.driverId
                if !route.driverId.isEmpty {
                    drivers.append(CashierDriverOption(
                        driverId: route.driverId,
                        label: route.driverId == driverDisplay ? route.driverId : "\(driverDisplay) (\(route.driverId))"
                    ))
                }
                if !effectiveRouteId.isEmpty {
                    routes.append(CashierRouteOption(
                        routeId: effectiveRouteId,
                        label: "\(effectiveRouteId) · \(driverDisplay) [\(route.manifestState)]",
                        linkedDriverId: route.driverId.isEmpty ? nil : route.driverId,
                        driverNameHint: route.driverName
                    ))
                }
            }
            return (drivers, routes)
        }
    ) {
        let token = TokenStore.shared.token
        let resolvedDepot = initialDepotId
            ?? TokenStore.shared.warehouseId
            ?? JwtPayload.homeNodeId(token: token)
            ?? ""
        let resolvedCell = initialCellId
            ?? MarketPackStore.current?.homeCell
            ?? JwtPayload.homeCell(token: token)
            ?? ""
        let resolvedSupplier = initialSupplierId
            ?? JwtPayload.supplierId(token: token)
            ?? ""

        self.cellId = resolvedCell
        self.depotId = resolvedDepot
        self.supplierId = resolvedSupplier
        self.startSessionHandler = startSessionHandler
        self.submitBlindCountHandler = submitBlindCountHandler
        self.reclassifyTenderHandler = reclassifyTenderHandler
        self.executeCITHandoverHandler = executeCITHandoverHandler
        self.checkAMLHandler = checkAMLHandler
        self.loadDriversHandler = loadDriversHandler
        self.loadManifestsHandler = loadManifestsHandler
        self.loadFleetLiveMapHandler = loadFleetLiveMapHandler
    }

    func setDenominationCount(noteSom: Int64, count: Int64) {
        let safe = max(0, count)
        denominations[noteSom] = safe
    }

    func adjustCount(noteSom: Int64, delta: Int64) {
        let current = denominations[noteSom] ?? 0
        setDenominationCount(noteSom: noteSom, count: current + delta)
    }

    func addStrap(noteSom: Int64, strapCount: Int64) {
        adjustCount(noteSom: noteSom, delta: strapCount)
    }

    func clearCounts() {
        for key in denominations.keys {
            denominations[key] = 0
        }
    }

    static func formatSom(amount: Int64) -> String {
        let safe = amount == 0 ? 0 : amount
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        formatter.groupingSeparator = " "
        formatter.usesGroupingSeparator = true
        formatter.maximumFractionDigits = 0
        let formatted = formatter.string(from: NSNumber(value: safe)) ?? "\(safe)"
        return "\(formatted) so'm"
    }

    static func formatWholeSom(minor: Int64) -> String {
        formatSom(amount: minor / 100)
    }

    func loadSelectors() async {
        isLoadingSelectors = true
        let warehouseScope = depotId.trimmingCharacters(in: .whitespacesAndNewlines)
        let scopedWarehouse: String? = warehouseScope.isEmpty ? nil : warehouseScope

        async let fetchedDrivers = try? loadDriversHandler()
        async let fetchedManifests = try? loadManifestsHandler()
        async let fetchedFleet = try? loadFleetLiveMapHandler(scopedWarehouse)

        let driversList = await fetchedDrivers ?? []
        let manifestsList = await fetchedManifests ?? []
        let (fleetDrivers, fleetRoutes) = await fetchedFleet ?? ([], [])

        var driversById: [String: CashierDriverOption] = [:]
        var orderedDrivers: [CashierDriverOption] = []
        for option in driversList + fleetDrivers where !option.driverId.isEmpty {
            if driversById[option.driverId] == nil {
                driversById[option.driverId] = option
                orderedDrivers.append(option)
            }
        }

        var routesById: [String: CashierRouteOption] = [:]
        var orderedRoutes: [CashierRouteOption] = []
        for route in fleetRoutes where !route.routeId.isEmpty {
            if routesById[route.routeId] == nil {
                routesById[route.routeId] = route
                orderedRoutes.append(route)
            }
        }
        for manifest in manifestsList where !manifest.routeId.isEmpty {
            if routesById[manifest.routeId] == nil {
                var matchedDriverId: String? = nil
                if let hint = manifest.driverNameHint?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased(),
                   !hint.isEmpty {
                    let matches = orderedDrivers.filter { $0.label.lowercased().hasPrefix(hint + " (") || $0.label.lowercased() == hint }
                    if matches.count == 1 {
                        matchedDriverId = matches[0].driverId
                    }
                }
                let option = CashierRouteOption(
                    routeId: manifest.routeId,
                    label: manifest.label,
                    linkedDriverId: matchedDriverId,
                    driverNameHint: manifest.driverNameHint
                )
                routesById[option.routeId] = option
                orderedRoutes.append(option)
            }
        }

        driverOptions = orderedDrivers
        routeOptions = orderedRoutes
        selectorsLoaded = true
        isLoadingSelectors = false

        if orderedDrivers.isEmpty && orderedRoutes.isEmpty {
            manualEntryMode = true
        }
    }

    func selectDriver(_ id: String) {
        driverId = id
        let trimmed = id.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        if routeId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty,
           let matchedRoute = routeOptions.first(where: { $0.linkedDriverId == trimmed }) {
            routeId = matchedRoute.routeId
        }
    }

    func selectRoute(_ id: String) {
        routeId = id
        let trimmed = id.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        if driverId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty,
           let matchedDriverId = routeOptions.first(where: { $0.routeId == trimmed })?.linkedDriverId {
            driverId = matchedDriverId
        }
    }

    func startSession(expectedCashMinor: Int64 = 0) async {
        let trimmedDepot = depotId.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedSupplier = supplierId.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedDriver = driverId.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedRoute = routeId.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedSeal = bagSealBarcode.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedCell = cellId.trimmingCharacters(in: .whitespacesAndNewlines)

        guard !trimmedDepot.isEmpty, !trimmedSupplier.isEmpty else {
            errorMessage = "Depot ID and Supplier ID are required before starting a cashier session."
            return
        }
        guard !trimmedDriver.isEmpty, !trimmedRoute.isEmpty, !trimmedSeal.isEmpty else {
            errorMessage = "Driver, Route, and Deposit Bag Seal Barcode are required."
            return
        }

        isLoading = true
        errorMessage = nil
        statusMessage = nil

        let req = StartCashierSessionRequest(
            cellId: trimmedCell,
            supplierId: trimmedSupplier,
            depotId: trimmedDepot,
            driverId: trimmedDriver,
            routeId: trimmedRoute,
            bagSealBarcode: trimmedSeal,
            expectedCashMinor: max(0, expectedCashMinor)
        )

        do {
            let sess = try await startSessionHandler(req)
            isLoading = false
            session = sess
            attemptNumber = sess.attemptNumber
            isBlindCountMasked = true
            isRecountRequired = false
            statusMessage = "Deposit Bag Registered. Please perform blind physical count."
        } catch let apiError as APIError {
            isLoading = false
            switch apiError {
            case .httpError(let code) where code == 409:
                errorMessage = "Bag Seal Conflict (409): Duplicate deposit bag seal barcode already logged."
            default:
                errorMessage = "Failed to start cashier session: \(apiError)"
            }
        } catch {
            isLoading = false
            errorMessage = error.localizedDescription
        }
    }

    func submitBlindCount(toleranceThresholdMinor: Int64 = 500000) async {
        guard let sessionId = session?.sessionId, !sessionId.isEmpty else {
            errorMessage = "Please start or select a session first"
            return
        }

        isLoading = true
        errorMessage = nil
        statusMessage = nil

        var stringDenoms: [String: Int64] = [:]
        for (note, count) in denominations {
            stringDenoms[String(note)] = count
        }
        let req = BlindCountRequest(
            denominationsUzs: stringDenoms,
            toleranceThresholdMinor: toleranceThresholdMinor
        )

        do {
            let updated = try await submitBlindCountHandler(sessionId, req)
            isLoading = false
            session = updated
            attemptNumber = updated.attemptNumber
            let isBal = updated.status == "BALANCED" || updated.status == "TOLERANCE_VARIANCE"
            let isQuar = updated.status == "DISCREPANT_QUARANTINE"

            isBalanced = isBal
            isQuarantined = isQuar
            isBlindCountMasked = !isBal && !isQuar
            isRecountRequired = false
            clearanceTokenHash = updated.clearanceTokenHash
            microSkimmingFlagged = updated.microSkimmingFlagged ?? false
            statusMessage = isBal ? "Count Balanced! Clearance Token Issued." : "Session quarantined for supervisor review."
        } catch let apiError as APIError {
            isLoading = false
            switch apiError {
            case .httpError(let code) where code == 409:
                isRecountRequired = true
                isBlindCountMasked = true
                attemptNumber = 2
                statusMessage = "MANDATORY BLIND RECOUNT #2 REQUIRED: Physical count does not match route records. Expected amount remains masked."
            default:
                errorMessage = "Count submission failed: \(apiError)"
            }
        } catch {
            isLoading = false
            errorMessage = error.localizedDescription
        }
    }

    @discardableResult
    func reclassifyTender(
        orderId: String,
        gatewayRrn: String,
        disputedCashMinor: Int64,
        paymentNetwork: String = "HUMO",
        terminalId: String = "",
        cardLastFour: String = "",
        authCode: String = ""
    ) async -> Bool {
        guard let sessionId = session?.sessionId, !sessionId.isEmpty else {
            errorMessage = "Active session required for tender reclassification"
            return false
        }

        let trimmedOrderId = orderId.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedRrn = gatewayRrn.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedOrderId.isEmpty, !trimmedRrn.isEmpty, disputedCashMinor > 0 else {
            errorMessage = "Order ID, Gateway RRN, and disputed amount (so'm) > 0 are required."
            return false
        }

        isLoading = true
        errorMessage = nil

        let effectiveTerminal = terminalId.isEmpty ? rrnTerminalId : terminalId
        let effectiveLastFour = cardLastFour.isEmpty ? rrnCardLastFour : cardLastFour
        let effectiveAuthCode = authCode.isEmpty ? rrnAuthCode : authCode
        let nowIso = Self.isoFormatter.string(from: Date())
        let req = RRNTenderReclassificationRequest(
            sessionId: sessionId,
            orderId: trimmedOrderId,
            disputedCashMinor: disputedCashMinor,
            gatewayRrn: trimmedRrn,
            terminalId: effectiveTerminal.trimmingCharacters(in: .whitespacesAndNewlines),
            cardLastFour: effectiveLastFour.trimmingCharacters(in: .whitespacesAndNewlines),
            authCode: effectiveAuthCode.trimmingCharacters(in: .whitespacesAndNewlines),
            paymentNetwork: paymentNetwork,
            rrnCapturedAt: nowIso,
            podGeofenceTimestamp: nowIso
        )

        do {
            let updated = try await reclassifyTenderHandler(req)
            isLoading = false
            session = updated
            isBalanced = updated.status == "BALANCED"
            clearanceTokenHash = updated.clearanceTokenHash
            statusMessage = "RRN Payment Reclassified! Compensating ledger applied."
            return true
        } catch {
            isLoading = false
            errorMessage = error.localizedDescription
            return false
        }
    }

    @discardableResult
    func executeCITHandover(
        courierBadgeId: String,
        citSealBarcode: String,
        amountMinor: Int64
    ) async -> Bool {
        let trimmedDepot = depotId.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedSupplier = supplierId.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedBadge = courierBadgeId.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedSeal = citSealBarcode.trimmingCharacters(in: .whitespacesAndNewlines)

        guard !trimmedDepot.isEmpty, !trimmedSupplier.isEmpty else {
            errorMessage = "Depot ID and Supplier ID are required for CIT handover."
            return false
        }
        guard !trimmedBadge.isEmpty, !trimmedSeal.isEmpty, amountMinor > 0 else {
            errorMessage = "Courier Badge ID, CIT Security Seal, and handover amount (so'm) > 0 are required."
            return false
        }

        isLoading = true
        errorMessage = nil

        let sessionIds = session.map { [$0.sessionId] } ?? []
        let req = CITHandoverRequest(
            depotId: trimmedDepot,
            supplierId: trimmedSupplier,
            sessionIds: sessionIds,
            courierBadgeId: trimmedBadge,
            citSealBarcode: trimmedSeal,
            cashMinor: amountMinor
        )

        do {
            let entry = try await executeCITHandoverHandler(req)
            isLoading = false
            lastVaultEntry = entry
            statusMessage = "CIT Armored Handover Complete: Manifest \(entry.citManifestId ?? entry.vaultEntryId) (\(Self.formatWholeSom(minor: entry.amountMinor)))"
            return true
        } catch {
            isLoading = false
            errorMessage = error.localizedDescription
            return false
        }
    }

    @discardableResult
    func checkAMLExposure(tin: String, cashMinor: Int64) async -> Bool {
        let trimmedTin = tin.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedSupplier = supplierId.trimmingCharacters(in: .whitespacesAndNewlines)

        guard !trimmedSupplier.isEmpty else {
            errorMessage = "Supplier ID is required for AML exposure check."
            return false
        }
        guard !trimmedTin.isEmpty, cashMinor > 0 else {
            errorMessage = "Legal Entity TIN and cash exposure amount (so'm) > 0 are required."
            return false
        }

        isLoading = true
        errorMessage = nil

        let today = Self.isoFormatter.string(from: Date())
        let req = AMLCheckRequest(
            legalEntityTin: trimmedTin,
            supplierId: trimmedSupplier,
            calendarDate: today,
            cashMinor: cashMinor
        )

        do {
            let resp = try await checkAMLHandler(req)
            isLoading = false
            amlExposureMinor = resp.accumulatedCashMinor
            amlCeilingMinor = resp.ceilingMinor
            amlExceeded = resp.ceilingMinor > 0 && resp.accumulatedCashMinor > resp.ceilingMinor
            statusMessage = resp.ceilingMinor > 0
                ? "AML Exposure OK: \(Self.formatWholeSom(minor: resp.accumulatedCashMinor)) / \(Self.formatWholeSom(minor: resp.ceilingMinor))"
                : "AML Exposure OK: \(Self.formatWholeSom(minor: resp.accumulatedCashMinor)) (Uncapped)"
            return true
        } catch let apiError as APIError {
            isLoading = false
            switch apiError {
            case .httpError(let code) where code == 409:
                amlExceeded = true
                errorMessage = "AML Ceiling Exceeded (409): Daily settlement exposure limit reached for TIN \(trimmedTin)."
            default:
                errorMessage = "AML check failed: \(apiError)"
            }
            return false
        } catch {
            isLoading = false
            errorMessage = error.localizedDescription
            return false
        }
    }
}
