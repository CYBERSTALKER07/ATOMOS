//
//  ExtractedDriverViewModelsTests.swift
//  driverappiosTests
//

import Foundation
import SwiftUI
import Testing
@testable import driverappios

private enum DriverTestMockError: LocalizedError {
    case backendFailure(String)

    var errorDescription: String? {
        switch self {
        case .backendFailure(let msg):
            return msg
        }
    }
}

/// Lock-protected mutable cell so `@Sendable` stub closures can share state with the test body.
private final class LockedBox<Value>: @unchecked Sendable {
    private let lock = NSLock()
    private var stored: Value

    init(_ value: Value) { stored = value }

    var value: Value {
        get { lock.withLock { stored } }
        set { lock.withLock { stored = newValue } }
    }
}

private final class StubFleetService: FleetServiceProtocol {
    var scanQRResult: Result<DeliveryScanQRResponse, Error>?
    var confirmOffloadResult: Result<ConfirmOffloadResponse, Error>?
    var completeOrderError: Error?
    var retryFiscalResult: Result<CollectCashResponse, Error>?
    var requestRescueError: Error?

    private(set) var scannedOrders: [(orderId: String, token: String)] = []
    private(set) var confirmedOrders: [String] = []
    private(set) var completedOrders: [String] = []
    private(set) var retriedFiscalOrders: [String] = []
    private(set) var rescueRequests: [(reason: String, note: String)] = []

    func fetchActiveMissions(routeId: String) async throws -> [Mission] { [] }
    func deliverOrder(orderId: String, scannedToken: String) async throws {}

    func validateQR(orderId: String, scannedToken: String) async throws -> ValidateQRResponse {
        ValidateQRResponse(orderId: orderId, retailerName: "Shop", totalAmount: 0, state: "ARRIVED", items: [])
    }

    func confirmOffload(orderId: String) async throws -> ConfirmOffloadResponse {
        confirmedOrders.append(orderId)
        if let confirmOffloadResult {
            return try confirmOffloadResult.get()
        }
        return ConfirmOffloadResponse(
            orderId: orderId,
            state: "AWAITING_PAYMENT",
            paymentMethod: "CASH",
            amount: 50_000,
            invoiceId: "INV-1",
            retailerId: "RET-1",
            message: "Collect payment"
        )
    }

    func scanDeliveryQR(orderId: String, qrToken: String) async throws -> DeliveryScanQRResponse {
        scannedOrders.append((orderId, qrToken))
        if let scanQRResult {
            return try scanQRResult.get()
        }
        let payload = Data("{\"valid\":true,\"order_id\":\"\(orderId)\",\"state\":\"AWAITING_PAYMENT\"}".utf8)
        return try JSONDecoder().decode(DeliveryScanQRResponse.self, from: payload)
    }

    func completeOrder(orderId: String) async throws {
        completedOrders.append(orderId)
        if let completeOrderError {
            throw completeOrderError
        }
    }

    func collectCash(orderId: String, amountReceivedMinor: Int64?) async throws -> CollectCashResponse {
        CollectCashResponse(
            orderId: orderId,
            state: "FISCALIZING",
            amount: 50_000,
            amountReceivedMinor: amountReceivedMinor.map(Int.init),
            shortfallMinor: 0,
            overageMinor: 0,
            distanceM: 5.0,
            message: "Captured",
            attemptId: "ATT-1",
            fiscalStatus: "PENDING",
            currency: "UZS"
        )
    }

    func retryFiscal(orderId: String) async throws -> CollectCashResponse {
        retriedFiscalOrders.append(orderId)
        if let retryFiscalResult {
            return try retryFiscalResult.get()
        }
        return CollectCashResponse(
            orderId: orderId,
            state: "FISCALIZING",
            amount: 50_000,
            amountReceivedMinor: 50_000,
            shortfallMinor: 0,
            overageMinor: 0,
            distanceM: 3.0,
            message: "Retrying",
            attemptId: "ATT-2",
            fiscalStatus: "RETRYING",
            currency: "UZS"
        )
    }

    func fetchOrderLineItems(orderId: String) async throws -> [LineItem] { [] }

    func amendOrder(
        orderId: String,
        driverId: String,
        items: [(lineItemId: String, rejectedQty: Int, status: LineItemStatus, reason: String, customReason: String?)]
    ) async throws {}

    func verifyHandshake(
        orderId: String,
        token: String,
        latitude: Double,
        longitude: Double
    ) async throws -> VerifyHandshakeResponse {
        VerifyHandshakeResponse(success: true, message: "OK")
    }

    func updateOrderDuringDelivery(
        orderId: String,
        latitude: Double,
        longitude: Double
    ) async throws -> UpdateOrderDuringDeliveryResponse {
        UpdateOrderDuringDeliveryResponse(success: true, message: "OK")
    }

    func markCreditDelivery(
        orderId: String,
        photoProofUrl: String?,
        signatureUrl: String?
    ) async throws -> [String: String] {
        ["status": "DELIVERED_ON_CREDIT"]
    }

    func splitPayment(
        orderId: String,
        cashMinor: Int64,
        cardMinor: Int64,
        currency: String?
    ) async throws -> SplitPaymentResponse {
        SplitPaymentResponse(status: "OK")
    }

    func requestRescue(reason: String, note: String) async throws {
        rescueRequests.append((reason, note))
        if let requestRescueError {
            throw requestRescueError
        }
    }

    func respondRescue(rescueId: String, brokenDriverId: String, accept: Bool) async throws {}
    func reassignHandshake(orderId: String) async throws {}
}

@MainActor
struct ExtractedDriverViewModelsTests {

    // MARK: - RootViewModel Tests

    @Test func rootViewModel_upToDateAndOutdatedAndForceDismissGuard() async {
        let upToDateVM = RootViewModel { _ in
            DriverClientPolicyResponse(
                outdated: false,
                forceUpdate: false,
                updateDeferred: false,
                minimumVersion: "1.0.0",
                recommendedVersion: "1.0.0",
                updateURL: "http://127.0.0.1:1/manifest.plist",
                deferReason: nil
            )
        }
        await upToDateVM.loadClientPolicy(currentVersion: "1.0.0")
        #expect(upToDateVM.clientPolicyMessage == nil)
        #expect(upToDateVM.clientPolicyForce == false)

        let softUpdateVM = RootViewModel { path in
            #expect(path.contains("platform=ios"))
            #expect(path.contains("version=1.0.0"))
            return DriverClientPolicyResponse(
                outdated: true,
                forceUpdate: false,
                updateDeferred: false,
                minimumVersion: "1.2.0",
                recommendedVersion: "1.3.0",
                updateURL: "http://127.0.0.1:1/manifest.plist",
                deferReason: nil
            )
        }
        await softUpdateVM.loadClientPolicy(currentVersion: "1.0.0")
        #expect(softUpdateVM.clientPolicyMessage?.contains("1.2.0") == true)
        #expect(softUpdateVM.clientPolicyForce == false)
        softUpdateVM.dismissBanner()
        #expect(softUpdateVM.clientPolicyMessage == nil)

        // Force update blocks dismissBanner
        let forcedVM = RootViewModel { _ in
            DriverClientPolicyResponse(
                outdated: true,
                forceUpdate: true,
                updateDeferred: false,
                minimumVersion: "2.0.0",
                recommendedVersion: "2.0.0",
                updateURL: "http://127.0.0.1:1/manifest.plist",
                deferReason: nil
            )
        }
        await forcedVM.loadClientPolicy(currentVersion: "1.0.0")
        #expect(forcedVM.clientPolicyForce == true)
        #expect(forcedVM.clientPolicyMessage?.contains("Update required") == true)
        forcedVM.dismissBanner()
        #expect(forcedVM.clientPolicyMessage != nil, "Forced policy banner must not be dismissible")
    }

    @Test func rootViewModel_networkFailureGracefullyIgnored() async {
        let vm = RootViewModel { _ in
            throw URLError(.cannotConnectToHost)
        }
        await vm.loadClientPolicy(currentVersion: "1.0.0")
        #expect(vm.clientPolicyMessage == nil)
        #expect(vm.clientPolicyForce == false)
    }

    // MARK: - DriverHomeViewModel Tests

    @Test func driverHomeViewModel_pulseHonestyHappyPathAndFallback() async throws {
        let shouldFail = LockedBox(false)
        let vm = DriverHomeViewModel {
            if shouldFail.value {
                throw URLError(.timedOut)
            }
            let json = Data("""
            {
              "events": [
                {
                  "id": "evt-1",
                  "kind": "DISPATCH",
                  "title": "Route Assigned",
                  "description": "Bay 4",
                  "occurred_at": "2026-11-12T08:00:00Z"
                }
              ],
              "fetched_at": "2026-11-12T08:00:01Z",
              "unread_count": 1
            }
            """.utf8)
            return try JSONDecoder().decode(PulseResponse.self, from: json)
        }

        await vm.loadPulse()
        #expect(vm.pulseLoading == false)
        #expect(vm.pulseError == nil)
        #expect(vm.pulseEvents.count == 1)
        #expect(vm.pulseEvents.first?.id == "evt-1")

        // Subsequent failure preserves previous pulse events and sets error
        shouldFail.value = true
        await vm.loadPulse()
        #expect(vm.pulseLoading == false)
        #expect(vm.pulseEvents.count == 1, "Previous pulse events must be retained on transient failure")
        #expect(vm.pulseError != nil)
    }

    @Test func driverHomeViewModel_handoffNavigationAndGreeting() {
        let vm = DriverHomeViewModel {
            let empty = Data("{\"events\":[],\"fetched_at\":\"\"}".utf8)
            return try JSONDecoder().decode(PulseResponse.self, from: empty)
        }

        var navigated: [HandoffDestination] = []
        vm.handleHandoffLink("/dispatch") { navigated.append($0) }
        vm.handleHandoffLink("/fleet") { navigated.append($0) }
        vm.handleHandoffLink("/manifests/M-101") { navigated.append($0) }
        #expect(navigated == [.home, .fleetMap, .manifestList])
        #expect(vm.showHandoffAlert == false)

        vm.handleHandoffLink("/unknown/portal-only") { navigated.append($0) }
        #expect(vm.showHandoffAlert == true)
        #expect(vm.handoffAlertMessage?.contains("no native route") == true)

        vm.clearHandoffAlert()
        #expect(vm.showHandoffAlert == false)
        #expect(vm.handoffAlertMessage == nil)

        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = TimeZone(secondsFromGMT: 0) ?? .current
        let morning = cal.date(from: DateComponents(year: 2026, month: 11, day: 12, hour: 8)) ?? Date()
        let afternoon = cal.date(from: DateComponents(year: 2026, month: 11, day: 12, hour: 14)) ?? Date()
        let evening = cal.date(from: DateComponents(year: 2026, month: 11, day: 12, hour: 19)) ?? Date()
        let night = cal.date(from: DateComponents(year: 2026, month: 11, day: 12, hour: 23)) ?? Date()

        #expect(vm.greetingText(for: morning, calendar: cal) == "GOOD MORNING")
        #expect(vm.greetingText(for: afternoon, calendar: cal) == "GOOD AFTERNOON")
        #expect(vm.greetingText(for: evening, calendar: cal) == "GOOD EVENING")
        #expect(vm.greetingText(for: night, calendar: cal) == "GOOD NIGHT")
    }

    // MARK: - RidesListViewModel Tests

    @Test func ridesListViewModel_lifoLoadLabelsAndEarlyComplete() async {
        let shouldFail = LockedBox(false)
        let vm = RidesListViewModel { reason, note in
            #expect(!reason.isEmpty)
            _ = note
            if shouldFail.value {
                throw DriverTestMockError.backendFailure("Supervisor rejected early end")
            }
        }

        // Loading mode off -> nil labels
        #expect(vm.loadSeqLabel(for: 0, total: 3) == nil)

        vm.loadingMode = true
        #expect(vm.loadSeqLabel(for: 0, total: 0) == nil)
        #expect(vm.loadSeqLabel(for: 0, total: 3) == "Load #1 · Back of Truck")
        #expect(vm.loadSeqLabel(for: 1, total: 3) == "Load #2")
        #expect(vm.loadSeqLabel(for: 2, total: 3) == "Load #3 · By the Doors")

        var reloadedCount = 0
        await vm.submitEarlyComplete(reason: "VEHICLE_ISSUE", note: "Brake warning") {
            reloadedCount += 1
        }
        #expect(vm.isRequestingEarlyComplete == false)
        #expect(vm.earlyCompleteMessage == "Early complete request submitted.")
        #expect(reloadedCount == 1)

        shouldFail.value = true
        await vm.submitEarlyComplete(reason: "VEHICLE_ISSUE", note: "Retry") {
            reloadedCount += 1
        }
        #expect(vm.earlyCompleteMessage == "Supervisor rejected early end")
        #expect(reloadedCount == 1)

        // In-flight reconnect recovery
        vm.isRequestingEarlyComplete = true
        await vm.handleReconnect { reloadedCount += 1 }
        #expect(vm.isRequestingEarlyComplete == false)
        #expect(vm.earlyCompleteMessage == DriverReconnectRecovery.hint)
        #expect(reloadedCount == 2)
    }

    // MARK: - OffloadReviewViewModel Tests

    @Test func offloadReviewViewModel_rejectionsValidationAndMinorUnitAdjustedTotal() async {
        let items = [
            OrderLineItem(productId: "SKU-1", productName: "Sunflower Oil 5L", quantity: 4, unitPrice: 85_000_00),
            OrderLineItem(productId: "SKU-2", productName: "Sugar 10kg", quantity: 2, unitPrice: 120_000_00),
        ]
        let qrResponse = ValidateQRResponse(
            orderId: "ORD-500",
            retailerName: "Baraka Market",
            totalAmount: 580_000_00,
            state: "ARRIVED",
            items: items
        )
        let fleet = StubFleetService()
        let reportedMissing = LockedBox<[MissingItemRequest]>([])

        let vm = OffloadReviewViewModel(
            response: qrResponse,
            scannedToken: "QR-TOK-500",
            fleetService: fleet,
            reportMissing: { orderId, missing, _ in
                #expect(orderId == "ORD-500")
                reportedMissing.value = missing
            }
        )

        // Bounds check on rejection stepper
        vm.decrementRejection(for: items[0])
        #expect(vm.hasRejections == false)

        for _ in 0..<6 {
            vm.incrementRejection(for: items[0])
        }
        #expect(vm.rejectedQty["SKU-1"] == 4, "Rejected quantity cannot exceed ordered quantity")
        #expect(vm.needsPhotoProof == true, "Default DAMAGED reason requires photo proof")

        // Decrement back to 1 rejected
        vm.decrementRejection(for: items[0])
        vm.decrementRejection(for: items[0])
        vm.decrementRejection(for: items[0])
        #expect(vm.rejectedQty["SKU-1"] == 1)

        // Missing photo guard for DAMAGED
        let blockedNoPhoto = await vm.confirmOffload()
        #expect(blockedNoPhoto == nil)
        #expect(vm.errorMessage?.contains("Photo required") == true)

        // OTHER reason without custom description guard
        vm.setReason(.OTHER, for: "SKU-1")
        let blockedEmptyOther = await vm.confirmOffload()
        #expect(blockedEmptyOther == nil)
        #expect(vm.errorMessage?.contains("Describe the issue") == true)

        // Provide custom description -> succeeds and computes exact integer minor-unit adjusted amount
        vm.setCustomReason("Torn outer carton", for: "SKU-1")
        let confirmed = await vm.confirmOffload()
        #expect(confirmed != nil)
        #expect(reportedMissing.value.count == 1)
        #expect(reportedMissing.value.first?.skuId == "SKU-1")
        #expect(reportedMissing.value.first?.missingQty == 1)
        // Accepted: 3 * 8,500,000 + 2 * 12,000,000 = 25,500,000 + 24,000,000 = 49,500,000 tiyin
        #expect(confirmed?.amount == 49_500_000)
        #expect(fleet.scannedOrders.count == 1)

        // Credit leave validation guards
        #expect(vm.validateCreditLeave() == nil)
        #expect(vm.errorMessage?.contains("PoD photo required") == true)
        vm.evidencePhotoURL = "https://cdn.pegasusx/pod-1.jpg"
        #expect(vm.validateCreditLeave() == nil)
        #expect(vm.errorMessage?.contains("Signature required") == true)
        vm.signatureLocalPath = "/tmp/sig-1.jpg"
        let creditProof = vm.validateCreditLeave()
        #expect(creditProof?.photoURL == "https://cdn.pegasusx/pod-1.jpg")
        #expect(creditProof?.sigLocal == "/tmp/sig-1.jpg")
    }

    @Test func offloadReviewViewModel_emptyTokenFallbackAndErrorPropagation() async {
        let qrResponse = ValidateQRResponse(
            orderId: "ORD-501",
            retailerName: "Minimarket",
            totalAmount: 50_000,
            state: "ARRIVED",
            items: []
        )
        let fleet = StubFleetService()
        let vm = OffloadReviewViewModel(response: qrResponse, scannedToken: "", fleetService: fleet)

        let res = await vm.confirmOffload()
        #expect(res?.orderId == "ORD-501")
        #expect(fleet.confirmedOrders == ["ORD-501"])

        fleet.confirmOffloadResult = .failure(DriverTestMockError.backendFailure("Order locked"))
        let failed = await vm.confirmOffload()
        #expect(failed == nil)
        #expect(vm.errorMessage == "Order locked")
    }

    // MARK: - ShopClosedWaitingViewModel Tests

    @Test func shopClosedWaitingViewModel_guardsOnlineOfflineBypassAndSocketEvents() async {
        DriverSocketState.shared.outdatedNotice = nil
        let reportCalls = LockedBox(0)
        let bypassShouldFail = LockedBox(false)

        let vm = ShopClosedWaitingViewModel(
            orderId: "ORD-777",
            driverId: "DRV-09",
            sendShopClosed: { orderId, photoURL, _ in
                #expect(orderId == "ORD-777")
                #expect(!photoURL.isEmpty)
                reportCalls.value += 1
            },
            sendBypass: { orderId, token in
                #expect(orderId == "ORD-777")
                #expect(token == "BYP-999")
                if bypassShouldFail.value {
                    throw DriverTestMockError.backendFailure("Invalid bypass token")
                }
            }
        )
        defer { vm.stopCountdown() }

        #expect(vm.countdownFormatted == "3:00")
        #expect(vm.statusIcon == "camera.fill")

        // Guard: cannot report without photo
        await vm.reportShopClosed()
        #expect(vm.reported == false)
        #expect(vm.reportError?.contains("Photo required") == true)

        // Online report with uploaded photo URL
        vm.photoURL = "https://cdn.pegasusx/closed-777.jpg"
        await vm.reportShopClosed()
        #expect(vm.reported == true)
        #expect(reportCalls.value == 1)
        #expect(vm.reportError == nil)
        vm.stopCountdown()

        // Offline fallback when only local photo exists
        let offlineVM = ShopClosedWaitingViewModel(
            orderId: "ORD-778",
            driverId: "DRV-09",
            sendShopClosed: { _, _, _ in
                throw URLError(.notConnectedToInternet)
            }
        )
        defer { offlineVM.stopCountdown() }
        offlineVM.photoLocalPath = "/tmp/local-closed.jpg"
        await offlineVM.reportShopClosed()
        #expect(offlineVM.reported == true)
        #expect(offlineVM.reportError?.contains("Offline") == true)
        offlineVM.stopCountdown()

        // Socket events for matching vs non-matching orderId
        vm.handleDriverSocketEvent(
            DriverSocketState.DriverEvent(type: "SHOP_CLOSED_RESPONSE", orderId: "OTHER-ORD", response: "Ignored")
        )
        #expect(vm.retailerResponse == nil)

        vm.handleDriverSocketEvent(
            DriverSocketState.DriverEvent(type: "SHOP_CLOSED_RESPONSE", orderId: "ORD-777", response: "Opening in 2 min")
        )
        #expect(vm.retailerResponse == "Opening in 2 min")
        #expect(vm.statusTitle == "Retailer Responded")

        vm.handleDriverSocketEvent(
            DriverSocketState.DriverEvent(type: "BYPASS_TOKEN_ISSUED", orderId: "ORD-777", bypassToken: "BYP-999")
        )
        #expect(vm.bypassToken == "BYP-999")

        vm.handleDriverSocketEvent(
            DriverSocketState.DriverEvent(type: "SHOP_CLOSED_ESCALATED", orderId: "ORD-777")
        )
        #expect(vm.isEscalated == true)

        // Bypass token submission happy path & error state
        vm.bypassInput = "BYP-999"
        let ok = await vm.submitBypass()
        #expect(ok == true)
        #expect(vm.bypassError == nil)

        bypassShouldFail.value = true
        let failed = await vm.submitBypass()
        #expect(failed == false)
        #expect(vm.bypassError == "Invalid bypass token")
    }

    // MARK: - PaymentWaitingViewModel Tests

    @Test func paymentWaitingViewModel_socketTransitionsFiscalRetryAndIdempotentComplete() async {
        DriverSocketState.shared.outdatedNotice = nil
        let fleet = StubFleetService()
        let vm = PaymentWaitingViewModel(
            orderId: "ORD-888",
            amount: 250_000_00,
            driverId: "DRV-11",
            fleetService: fleet
        )

        #expect(vm.headerTitle == "Awaiting Payment")

        // Cash collection redirect event
        vm.handleDriverSocketEvent(
            DriverSocketState.DriverEvent(
                type: "ORDER_STATUS_CHANGED",
                orderId: "ORD-888",
                status: "PENDING_CASH_COLLECTION"
            )
        )
        #expect(vm.requiresCashCollection == true)

        // Finalize delivery happy path -> enters fiscalizing
        await vm.finalizeDelivery()
        #expect(fleet.completedOrders == ["ORD-888"])
        #expect(vm.isSettled == true)
        #expect(vm.isFiscalizing == true)
        #expect(vm.headerTitle == "Fiscalizing")

        // Fiscal failure socket event -> retry fiscal
        vm.handleDriverSocketEvent(
            DriverSocketState.DriverEvent(type: "FISCAL_RECEIPT_FAILED", orderId: "ORD-888")
        )
        #expect(vm.isFiscalFailed == true)
        #expect(vm.isFiscalizing == false)
        #expect(vm.headerTitle == "Fiscal Failed")

        fleet.retryFiscalResult = .failure(DriverTestMockError.backendFailure("Fiscal module busy"))
        await vm.retryFiscal()
        #expect(vm.isFiscalFailed == true)
        #expect(vm.errorMessage == "Fiscal module busy")

        fleet.retryFiscalResult = nil
        await vm.retryFiscal()
        #expect(vm.isFiscalizing == true)
        #expect(vm.isFiscalFailed == false)

        // Order completed socket event
        vm.handleDriverSocketEvent(
            DriverSocketState.DriverEvent(type: "ORDER_COMPLETED", orderId: "ORD-888")
        )
        #expect(vm.isDone == true)

        // Idempotent "already COMPLETED" backend error treated as done
        let idempotentVM = PaymentWaitingViewModel(
            orderId: "ORD-889",
            amount: 10_000,
            driverId: "DRV-11",
            fleetService: fleet
        )
        fleet.completeOrderError = DriverTestMockError.backendFailure("Order is already COMPLETED")
        await idempotentVM.finalizeDelivery()
        #expect(idempotentVM.isDone == true)
        #expect(idempotentVM.errorMessage == nil)
    }

    // MARK: - FleetMapViewModel, RequestRescueViewModel & LoginViewModel Tests

    @Test func fleetMapAndRescueAndLoginSanitization() async {
        let mapVM = FleetMapViewModel()
        mapVM.phase = .activeDelivery
        let qr = ValidateQRResponse(orderId: "ORD-1", retailerName: "Shop", totalAmount: 1000, state: "ARRIVED", items: [])
        mapVM.handleValidatedQR(qr, token: "TOK-1")
        #expect(mapVM.scannedQRToken == "TOK-1")
        #expect(mapVM.navPath.count == 1)

        let offload = ConfirmOffloadResponse(
            orderId: "ORD-1",
            state: "AWAITING_PAYMENT",
            paymentMethod: "CASH",
            amount: 1000,
            invoiceId: nil,
            retailerId: "R-1",
            message: "OK"
        )
        mapVM.handleConfirmedOffload(offload)
        #expect(mapVM.navPath.count == 2)

        mapVM.resetNavigationAndReturnToPicker()
        #expect(mapVM.navPath.count == 0)
        #expect(mapVM.phase == .pickingOrder)
        #expect(mapVM.selectedMission == nil)

        // RequestRescueViewModel happy + error paths
        let fleet = StubFleetService()
        let rescueVM = RequestRescueViewModel(fleetService: fleet)
        rescueVM.reason = "Flat Tire"
        rescueVM.note = "Rear left tire punctured"
        let ok = await rescueVM.submitRescue()
        #expect(ok == true)
        #expect(fleet.rescueRequests.first?.reason == "Flat Tire")
        #expect(rescueVM.errorMessage == nil)

        fleet.requestRescueError = DriverTestMockError.backendFailure("Dispatch unreachable")
        let failed = await rescueVM.submitRescue()
        #expect(failed == false)
        #expect(rescueVM.errorMessage == "Dispatch unreachable")

        // LoginViewModel sanitization & mode reset
        let loginVM = LoginViewModel()
        loginVM.setPin("12a34-56789")
        #expect(loginVM.pin == "123456")
        loginVM.setOtp("99 88 77 66")
        #expect(loginVM.otpCode == "998877")
        loginVM.setMode(.pinDev)
        #expect(loginVM.pin.isEmpty)
        #expect(loginVM.otpCode.isEmpty)
    }
}
