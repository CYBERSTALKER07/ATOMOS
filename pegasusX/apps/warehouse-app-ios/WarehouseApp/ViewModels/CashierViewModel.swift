//
//  CashierViewModel.swift
//  WarehouseAppIOS
//

import Foundation
import SwiftUI
import Observation

struct DenominationRowItem: Identifiable, Sendable {
    var id: Int64 { noteUzs }
    let noteUzs: Int64
    var count: Int64

    var totalUzs: Int64 {
        noteUzs * count
    }

    var totalTiyin: Int64 {
        noteUzs * count * 100
    }
}

@Observable
@MainActor
final class CashierViewModel {
    private static let isoFormatter = ISO8601DateFormatter()

    var depotId: String = "depot-tashkent-south"
    var supplierId: String = "supplier-tashkent-1"
    var driverId: String = "driver-uz-09"
    var routeId: String = "route-tashkent-042"
    var bagSealBarcode: String = "BAG-UZ-99120"

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

    var rrnOrderId: String = "order-uz-8812"
    var rrnGateway: String = "123456789012"
    var rrnDisputedText: String = "500000"

    var citBadgeId: String = "INKAS-9921"
    var citSealBarcode: String = "CIT-SEAL-7712"
    var citHandoverText: String = "1425000"

    var amlTin: String = "123456789"
    var amlExposureText: String = "15000000"

    var totalBanknotesCount: Int64 {
        denominations.values.reduce(0, +)
    }

    private let startSessionHandler: @Sendable (StartCashierSessionRequest) async throws -> CashierSessionView
    private let submitBlindCountHandler: @Sendable (String, BlindCountRequest) async throws -> CashierSessionView
    private let reclassifyTenderHandler: @Sendable (RRNTenderReclassificationRequest) async throws -> CashierSessionView
    private let executeCITHandoverHandler: @Sendable (CITHandoverRequest) async throws -> DepotVaultLedgerEntry
    private let checkAMLHandler: @Sendable (AMLCheckRequest) async throws -> AMLCheckResponse

    init(
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
        }
    ) {
        self.startSessionHandler = startSessionHandler
        self.submitBlindCountHandler = submitBlindCountHandler
        self.reclassifyTenderHandler = reclassifyTenderHandler
        self.executeCITHandoverHandler = executeCITHandoverHandler
        self.checkAMLHandler = checkAMLHandler
    }

    func setDenominationCount(noteUzs: Int64, count: Int64) {
        let safe = max(0, count)
        denominations[noteUzs] = safe
    }

    func adjustCount(noteUzs: Int64, delta: Int64) {
        let current = denominations[noteUzs] ?? 0
        setDenominationCount(noteUzs: noteUzs, count: current + delta)
    }

    func addStrap(noteUzs: Int64, strapCount: Int64) {
        adjustCount(noteUzs: noteUzs, delta: strapCount)
    }

    func clearCounts() {
        for key in denominations.keys {
            denominations[key] = 0
        }
    }

    func quickFillSampleBalanced() {
        clearCounts()
        setDenominationCount(noteUzs: 200000, count: 7)
        setDenominationCount(noteUzs: 20000, count: 1)
        setDenominationCount(noteUzs: 5000, count: 1)
    }

    func quickFillSampleDiscrepant() {
        clearCounts()
        setDenominationCount(noteUzs: 200000, count: 6)
    }

    static func formatUzs(amount: Int64) -> String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        formatter.groupingSeparator = " "
        let formatted = formatter.string(from: NSNumber(value: amount)) ?? "\(amount)"
        return "\(formatted) UZS"
    }

    func startSession(expectedCashMinor: Int64 = 1425000000) async {
        isLoading = true
        errorMessage = nil
        statusMessage = nil

        let req = StartCashierSessionRequest(
            cellId: "cell-uz",
            supplierId: supplierId,
            depotId: depotId,
            driverId: driverId,
            routeId: routeId,
            bagSealBarcode: bagSealBarcode,
            expectedCashMinor: expectedCashMinor
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

    func reclassifyTender(
        orderId: String,
        gatewayRrn: String,
        disputedCashMinor: Int64,
        paymentNetwork: String = "HUMO"
    ) async {
        guard let sessionId = session?.sessionId, !sessionId.isEmpty else {
            errorMessage = "Active session required for tender reclassification"
            return
        }

        isLoading = true
        errorMessage = nil

        let nowIso = Self.isoFormatter.string(from: Date())
        let req = RRNTenderReclassificationRequest(
            sessionId: sessionId,
            orderId: orderId,
            disputedCashMinor: disputedCashMinor,
            gatewayRrn: gatewayRrn,
            authCode: "AUTH-\(Int.random(in: 100000...999999))",
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
        } catch {
            isLoading = false
            errorMessage = error.localizedDescription
        }
    }

    func executeCITHandover(
        courierBadgeId: String,
        citSealBarcode: String,
        amountMinor: Int64
    ) async {
        let sessId = session?.sessionId ?? "sess-batch-01"
        isLoading = true
        errorMessage = nil

        let req = CITHandoverRequest(
            depotId: depotId,
            supplierId: supplierId,
            sessionIds: [sessId],
            courierBadgeId: courierBadgeId,
            citSealBarcode: citSealBarcode,
            cashMinor: amountMinor
        )

        do {
            let entry = try await executeCITHandoverHandler(req)
            isLoading = false
            lastVaultEntry = entry
            statusMessage = "CIT Armored Handover Complete: Manifest \(entry.citManifestId ?? entry.vaultEntryId)"
        } catch {
            isLoading = false
            errorMessage = error.localizedDescription
        }
    }

    func checkAMLExposure(tin: String, cashMinor: Int64) async {
        isLoading = true
        errorMessage = nil

        let today = Self.isoFormatter.string(from: Date())
        let req = AMLCheckRequest(
            legalEntityTin: tin,
            supplierId: supplierId,
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
                ? "AML Exposure OK: \(resp.accumulatedCashMinor / 100) / \(resp.ceilingMinor / 100) UZS"
                : "AML Exposure OK: \(resp.accumulatedCashMinor / 100) UZS (Uncapped)"
        } catch let apiError as APIError {
            isLoading = false
            switch apiError {
            case .httpError(let code) where code == 409:
                amlExceeded = true
                errorMessage = "AML Ceiling Exceeded (409): Daily settlement exposure limit reached for TIN \(tin)."
            default:
                errorMessage = "AML check failed: \(apiError)"
            }
        } catch {
            isLoading = false
            errorMessage = error.localizedDescription
        }
    }
}
