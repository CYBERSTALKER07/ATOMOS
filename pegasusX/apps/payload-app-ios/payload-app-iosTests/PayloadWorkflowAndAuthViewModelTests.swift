//
//  PayloadWorkflowAndAuthViewModelTests.swift
//  payload-app-iosTests
//

import XCTest
@testable import payload_app_ios

@MainActor
final class PayloadWorkflowAndAuthViewModelTests: XCTestCase {

    // MARK: - RootViewModel Tests

    func testRootViewModelUpToDatePolicyClearsBanner() async {
        var installPrompted = false
        let vm = RootViewModel(
            fetchPolicy: { _, _ in
                ClientPolicyResponse(
                    role: "PAYLOAD",
                    outdated: false,
                    forceUpdate: false,
                    updateDeferred: false,
                    minimumVersion: "1.0.0",
                    recommendedVersion: "1.0.0",
                    updateURL: nil,
                    deferReason: nil
                )
            },
            evaluatePolicy: { _ in
                AutoUpdater.UpdateState(
                    available: false,
                    force: false,
                    deferred: false,
                    message: nil,
                    manifest: nil
                )
            },
            promptInstall: { _, _ in installPrompted = true }
        )

        await vm.loadClientPolicy(versionOverride: "1.2.0")

        XCTAssertNil(vm.clientPolicyMessage)
        XCTAssertFalse(vm.clientPolicyForce)
        XCTAssertFalse(installPrompted)
    }

    func testRootViewModelOptionalUpdateAllowsDismissAndManualInstall() async {
        var promptedForce: Bool?
        let manifest = AutoUpdater.Manifest(
            versionCode: 42,
            versionName: "2.1.0",
            manifestURL: URL(fileURLWithPath: "/tmp/manifest.plist"),
            notes: "Performance hardening"
        )
        let vm = RootViewModel(
            fetchPolicy: { _, _ in
                ClientPolicyResponse(
                    role: "PAYLOAD",
                    outdated: true,
                    forceUpdate: false,
                    updateDeferred: false,
                    minimumVersion: "1.0.0",
                    recommendedVersion: "2.1.0",
                    updateURL: nil,
                    deferReason: nil
                )
            },
            evaluatePolicy: { _ in
                AutoUpdater.UpdateState(
                    available: true,
                    force: false,
                    deferred: false,
                    message: "Update available (recommended 2.1.0)",
                    manifest: manifest
                )
            },
            promptInstall: { _, force in promptedForce = force }
        )

        await vm.loadClientPolicy(versionOverride: "1.0.0")

        XCTAssertEqual(vm.clientPolicyMessage, "Update available (recommended 2.1.0)")
        XCTAssertFalse(vm.clientPolicyForce)
        XCTAssertNil(promptedForce)

        vm.triggerInstall()
        XCTAssertEqual(promptedForce, false)

        vm.dismissBanner()
        XCTAssertNil(vm.clientPolicyMessage)
    }

    func testRootViewModelForceUpdateAutoPromptsAndBlocksDismiss() async {
        var autoPromptedForce: Bool?
        let vm = RootViewModel(
            fetchPolicy: { _, _ in
                ClientPolicyResponse(
                    role: "PAYLOAD",
                    outdated: true,
                    forceUpdate: true,
                    updateDeferred: false,
                    minimumVersion: "3.0.0",
                    recommendedVersion: "3.0.0",
                    updateURL: nil,
                    deferReason: nil
                )
            },
            evaluatePolicy: { _ in
                AutoUpdater.UpdateState(
                    available: true,
                    force: true,
                    deferred: false,
                    message: "Update required — minimum version 3.0.0",
                    manifest: nil
                )
            },
            promptInstall: { _, force in autoPromptedForce = force }
        )

        await vm.loadClientPolicy(versionOverride: "2.0.0")

        XCTAssertEqual(vm.clientPolicyMessage, "Update required — minimum version 3.0.0")
        XCTAssertTrue(vm.clientPolicyForce)
        XCTAssertEqual(autoPromptedForce, true)

        // Forced updates must never be dismissible
        vm.dismissBanner()
        XCTAssertEqual(vm.clientPolicyMessage, "Update required — minimum version 3.0.0")
    }

    func testRootViewModelNetworkFailureFailsOpenWithoutBlockingShell() async {
        let vm = RootViewModel(
            fetchPolicy: { _, _ in throw APIError.networkError },
            evaluatePolicy: { _ in
                XCTFail("evaluatePolicy should not be called when fetchPolicy throws")
                return AutoUpdater.UpdateState(available: false, force: false, deferred: false, message: nil, manifest: nil)
            },
            promptInstall: { _, _ in XCTFail("promptInstall should not be called on network error") }
        )

        await vm.loadClientPolicy(versionOverride: "1.0.0")
        XCTAssertNil(vm.clientPolicyMessage)
        XCTAssertFalse(vm.clientPolicyForce)
    }

    // MARK: - LoginViewModel Sanitization, Validation & Auth Flows

    func testLoginViewModelSanitizesOtpAndPinAndResetsOnModeSwitch() {
        let vm = LoginViewModel()
        vm.setOtp(" 98a-76 54 321 ")
        XCTAssertEqual(vm.otpCode, "987654")

        vm.setPin("pin:1234-5678-90")
        XCTAssertEqual(vm.pin, "12345678")

        vm.phone = " +998901234567 "
        XCTAssertTrue(vm.canSubmitPin)

        vm.setMode(.pinDev)
        XCTAssertEqual(vm.mode, .pinDev)
        XCTAssertEqual(vm.otpCode, "")
        XCTAssertEqual(vm.pin, "")
        XCTAssertFalse(vm.otpSent)
        XCTAssertFalse(vm.canSubmitPin)
    }

    func testLoginViewModelOtpHappyPathAndValidation() async {
        var sentPhone: String?
        var authenticatedWorker: String?
        let vm = LoginViewModel(
            sendPhoneVerification: { phone in sentPhone = phone },
            verifySmsCode: { code in "firebase-id-\(code)" },
            loginWithIdToken: { idToken in
                XCTAssertEqual(idToken, "firebase-id-654321")
                return LoginResponse(
                    token: "jwt-1",
                    refreshToken: "ref-1",
                    workerId: "wrk-77",
                    supplierId: "sup-1",
                    role: "PAYLOAD",
                    name: "Aziz",
                    warehouseId: "wh-1",
                    warehouseName: "Tashkent Hub"
                )
            },
            onAuthenticated: { resp in authenticatedWorker = resp.workerId }
        )

        // Blank phone validation
        vm.phone = "   "
        await vm.sendOtp()
        XCTAssertEqual(vm.error, "Phone number required")
        XCTAssertFalse(vm.otpSent)

        // Valid sendOtp
        vm.phone = " +998901112233 "
        await vm.sendOtp()
        XCTAssertNil(vm.error)
        XCTAssertTrue(vm.otpSent)
        XCTAssertEqual(sentPhone, "+998901112233")

        // Short OTP validation
        vm.otpCode = "123"
        await vm.verifyOtp()
        XCTAssertEqual(vm.error, "Enter the 6-digit verification code")

        // Valid 6-digit OTP
        vm.otpCode = "654-321"
        await vm.verifyOtp()
        XCTAssertNil(vm.error)
        XCTAssertEqual(authenticatedWorker, "wrk-77")
    }

    func testLoginViewModelPinAuthErrorsAndSuccess() async {
        var shouldFailUnauthorized = true
        var savedRole: String?

        let vm = LoginViewModel(
            loginWithPin: { phone, pin in
                if shouldFailUnauthorized {
                    throw APIError.unauthorized
                }
                return LoginResponse(
                    token: "jwt-pin",
                    refreshToken: nil,
                    workerId: "wrk-12",
                    supplierId: "sup-1",
                    role: "PAYLOAD",
                    name: "Timur",
                    warehouseId: "wh-2",
                    warehouseName: "Samarkand Hub"
                )
            },
            onAuthenticated: { resp in savedRole = resp.role }
        )

        vm.setMode(.pinDev)
        vm.phone = "+998901112233"
        vm.setPin("12345") // only 5 digits
        await vm.submitPin()
        XCTAssertEqual(vm.error, "Phone and PIN required")

        vm.setPin("123456")
        await vm.submitPin()
        XCTAssertEqual(vm.error, "Invalid credentials")
        XCTAssertNil(savedRole)

        shouldFailUnauthorized = false
        await vm.submitPin()
        XCTAssertNil(vm.error)
        XCTAssertEqual(savedRole, "PAYLOAD")
    }

    // MARK: - HomeViewModel Checklist, Countdown, Banners, Offline Queue & Notifications

    func testHomeViewModelChecklistSealingEligibilityAndCountdownAdvance() {
        let vm = HomeViewModel(queue: OfflineQueue(inMemory: true))
        let order1 = LiveOrder(
            orderId: "ord-1001",
            retailerId: "ret-1",
            amount: 500_000,
            paymentGateway: "PAYME",
            state: "LOADED",
            routeId: "rt-1",
            warehouseId: "wh-1",
            items: [
                LiveOrderItem(lineItemId: "li-a", skuId: "sku-a", skuName: "Cola", quantity: 2, unitPrice: 100_000, status: nil),
                LiveOrderItem(lineItemId: "li-b", skuId: "sku-b", skuName: "Fanta", quantity: 3, unitPrice: 100_000, status: nil),
            ]
        )
        let order2 = LiveOrder(
            orderId: "ord-1002",
            retailerId: "ret-2",
            amount: 200_000,
            paymentGateway: "CASH",
            state: "LOADED",
            routeId: "rt-1",
            warehouseId: "wh-1",
            items: [
                LiveOrderItem(lineItemId: "li-c", skuId: "sku-c", skuName: "Juice", quantity: 2, unitPrice: 100_000, status: nil),
            ]
        )
        let emptyOrder = LiveOrder(
            orderId: "ord-empty",
            retailerId: "ret-3",
            amount: 0,
            paymentGateway: nil,
            state: "LOADED",
            routeId: "rt-1",
            warehouseId: "wh-1",
            items: []
        )

        vm.hydrateState(
            trucks: [Truck.fixture(id: "trk-1", truckStatus: "LOADING")],
            selectedTruckId: "trk-1",
            manifest: Manifest(manifestId: "mf-1", truckId: "trk-1", state: "LOADING"),
            orders: [order1, order2, emptyOrder],
            selectedOrderId: "ord-1001",
            sealedOrderIds: ["ord-1001"],
            dispatchCodes: ["ord-1001": "DSP-901"],
            postSealOrderId: "ord-1001",
            postSealCountdown: 45
        )

        // Already sealed order and empty order can never be sealed
        XCTAssertFalse(vm.canSealOrder("ord-1001"))
        XCTAssertFalse(vm.canSealOrder("ord-empty"))
        XCTAssertFalse(vm.canSealOrder("ord-1002"))
        XCTAssertFalse(vm.allOrdersSealed)

        // Dismissing countdown clears postSealOrderId and advances to first unsealed order (ord-1002)
        vm.dismissCountdown()
        XCTAssertNil(vm.postSealOrderId)
        XCTAssertEqual(vm.postSealCountdown, 0)
        XCTAssertEqual(vm.selectedOrderId, "ord-1002")

        // Checking item on ord-1002 enables sealing, unchecking disables it
        vm.toggleItem("li-c")
        XCTAssertTrue(vm.canSealOrder("ord-1002"))
        vm.toggleItem("li-c")
        XCTAssertFalse(vm.canSealOrder("ord-1002"))
    }

    func testHomeViewModelOfflineOrderInjectionEnqueuesAndAutoDismissesBanner() async throws {
        let queue = OfflineQueue(inMemory: true)
        let vm = HomeViewModel(
            queue: queue,
            bannerAutoDismissNanoseconds: 30_000_000 // 30ms for fast deterministic test
        )
        vm.hydrateState(
            trucks: [Truck.fixture(id: "trk-9", truckStatus: "LOADING")],
            selectedTruckId: "trk-9",
            manifest: Manifest(manifestId: "mf-99", truckId: "trk-9", state: "LOADING"),
            orders: [],
            online: false
        )

        await vm.injectOrder("  ord-777  ")

        XCTAssertEqual(vm.queuedActions, 1)
        XCTAssertEqual(vm.queuedNoticeMessage, "Queued offline. Will sync when connection restores.")
        let actions = queue.read()
        XCTAssertEqual(actions.count, 1)
        XCTAssertEqual(actions.first?.id, "payload-inject-order-mf-99-ord-777")
        XCTAssertEqual(actions.first?.endpoint, "v1/supplier/manifests/mf-99/inject-order")

        // Wait for auto-dismiss timer to fire inside HomeViewModel
        try await Task.sleep(nanoseconds: 90_000_000)
        XCTAssertNil(vm.queuedNoticeMessage)
    }

    func testHomeViewModelHandoffLinksAndNotificationReadIdempotency() async {
        let vm = HomeViewModel(queue: OfflineQueue(inMemory: true))
        let orderOnTruck = LiveOrder(
            orderId: "ord-live-1",
            retailerId: "ret-1",
            amount: 100_000,
            paymentGateway: nil,
            state: "LOADED",
            routeId: nil,
            warehouseId: nil,
            items: []
        )
        let n1 = NotificationItem(
            notificationId: "n-1",
            type: "ALERT",
            title: "Dock Ready",
            body: "Bay 4 open",
            payload: nil,
            channel: "ops",
            readAt: nil,
            createdAt: "2026-11-12T10:00:00Z",
            handoffMetadata: nil
        )
        let n2 = NotificationItem(
            notificationId: "n-2",
            type: "INFO",
            title: "Shift Sync",
            body: "Manifest updated",
            payload: nil,
            channel: "ops",
            readAt: nil,
            createdAt: "2026-11-12T10:05:00Z",
            handoffMetadata: nil
        )

        vm.hydrateState(
            trucks: [Truck.fixture(id: "trk-1", truckStatus: "LOADING")],
            selectedTruckId: "trk-1",
            manifest: Manifest(manifestId: "mf-1", truckId: "trk-1", state: "LOADING"),
            orders: [orderOnTruck],
            notifications: [n1, n2],
            online: true
        )
        XCTAssertEqual(vm.unreadCount, 2)

        // Marking n-1 read twice only decrements unreadCount once
        vm.markNotificationRead("n-1")
        XCTAssertEqual(vm.unreadCount, 1)
        vm.markNotificationRead("n-1")
        XCTAssertEqual(vm.unreadCount, 1)

        vm.markAllNotificationsRead()
        XCTAssertEqual(vm.unreadCount, 0)
        XCTAssertTrue(vm.notifications.allSatisfy { !$0.isUnread })

        // Handoff link to order on current truck selects it without alert
        vm.showNotificationsPanel = true
        await vm.handleHandoffLink("/orders/ord-live-1")
        XCTAssertFalse(vm.showNotificationsPanel)
        XCTAssertEqual(vm.selectedOrderId, "ord-live-1")
        XCTAssertFalse(vm.isShowingHandoffAlert)

        // Handoff link to order NOT on current truck triggers alert
        await vm.handleHandoffLink("/orders/ord-other-truck")
        XCTAssertTrue(vm.isShowingHandoffAlert)
        XCTAssertEqual(vm.handoffNavigationMessage, "Open in portal — order not on this truck")

        vm.clearHandoffNavigationMessage()
        XCTAssertFalse(vm.isShowingHandoffAlert)
        XCTAssertNil(vm.handoffNavigationMessage)
    }

    func testInjectOrderViewModelBarcodeTrimmingAndSubmissionGate() {
        let vm = InjectOrderViewModel()
        XCTAssertFalse(vm.canSubmit)

        vm.toggleScanner()
        XCTAssertTrue(vm.showScanner)

        vm.applyScannedBarcode("  ORD-88219 \n")
        XCTAssertFalse(vm.showScanner)
        XCTAssertEqual(vm.orderId, "ORD-88219")
        XCTAssertEqual(vm.trimmedOrderId, "ORD-88219")
        XCTAssertTrue(vm.canSubmit)
    }
}
