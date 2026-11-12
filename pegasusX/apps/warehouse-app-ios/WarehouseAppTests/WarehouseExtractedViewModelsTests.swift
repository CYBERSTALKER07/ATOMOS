//
//  WarehouseExtractedViewModelsTests.swift
//  WarehouseAppTests
//

import CoreLocation
import XCTest
@testable import WarehouseAppIOS

@MainActor
final class WarehouseExtractedViewModelsTests: XCTestCase {

    override func tearDown() {
        TokenStore.shared.clear()
        super.tearDown()
    }

    private func makeJwt(homeNodeId: String?, isConfigured: Bool) throws -> String {
        var payload: [String: Any] = ["is_configured": isConfigured]
        if let homeNodeId {
            payload["home_node_id"] = homeNodeId
        }
        let headerData = try JSONSerialization.data(withJSONObject: ["alg": "HS256", "typ": "JWT"])
        let payloadData = try JSONSerialization.data(withJSONObject: payload)
        let h = headerData.base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
        let p = payloadData.base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
        return "\(h).\(p).sig"
    }

    // MARK: - RootViewModel

    func testRootViewModelDismissibleVsForceUpdatePolicy() async throws {
        let manifestURL = try XCTUnwrap(URL(string: "https://pegasusx.app/manifest.plist"))
        let sampleManifest = AutoUpdater.Manifest(
            versionCode: 42,
            versionName: "2.0.0",
            manifestURL: manifestURL,
            notes: "Security patch"
        )

        let optionalVm = RootViewModel(
            fetchPolicyHandler: { version in
                XCTAssertEqual(version, "1.2.0")
                let raw = try XCTUnwrap("""
                {
                  "outdated": true,
                  "force_update": false,
                  "minimum_version": "1.1.0",
                  "recommended_version": "2.0.0"
                }
                """.data(using: .utf8))
                return try JSONDecoder().decode(WarehouseClientPolicyResponse.self, from: raw)
            },
            evaluatePolicyHandler: { _ in
                AutoUpdater.UpdateState(
                    available: true,
                    force: false,
                    deferred: false,
                    message: "Update available — minimum version 1.1.0",
                    manifest: sampleManifest
                )
            }
        )

        await optionalVm.loadClientPolicy(versionOverride: "1.2.0")
        XCTAssertEqual(optionalVm.clientPolicyMessage, "Update available — minimum version 1.1.0")
        XCTAssertFalse(optionalVm.clientPolicyForce)
        XCTAssertEqual(optionalVm.pendingManifest?.versionCode, 42)

        optionalVm.dismissBanner()
        XCTAssertNil(optionalVm.clientPolicyMessage)

        let forcedVm = RootViewModel(
            fetchPolicyHandler: { _ in
                let raw = try XCTUnwrap("""
                {
                  "outdated": true,
                  "force_update": true,
                  "minimum_version": "2.0.0"
                }
                """.data(using: .utf8))
                return try JSONDecoder().decode(WarehouseClientPolicyResponse.self, from: raw)
            },
            evaluatePolicyHandler: { _ in
                AutoUpdater.UpdateState(
                    available: false,
                    force: true,
                    deferred: false,
                    message: "Update required — minimum version 2.0.0",
                    manifest: nil
                )
            }
        )

        await forcedVm.loadClientPolicy(versionOverride: "1.0.0")
        XCTAssertTrue(forcedVm.clientPolicyForce)
        XCTAssertEqual(forcedVm.clientPolicyMessage, "Update required — minimum version 2.0.0")
        forcedVm.dismissBanner()
        XCTAssertEqual(forcedVm.clientPolicyMessage, "Update required — minimum version 2.0.0")
    }

    // MARK: - LoginViewModel

    func testLoginViewModelOtpFlowSuccessAnd404WarehouseError() async throws {
        var sentPhone: String?
        var uploadedPush = false
        var shouldReturn404 = true
        let validToken = try makeJwt(homeNodeId: "wh-tashkent-01", isConfigured: true)

        let vm = LoginViewModel(
            configureAuthHandler: {},
            resetFlowHandler: {},
            sendOtpHandler: { phone in
                sentPhone = phone
            },
            verifySmsHandler: { code in
                XCTAssertEqual(code, "123456")
                return "firebase-id-token"
            },
            loginIdTokenHandler: { idToken in
                XCTAssertEqual(idToken, "firebase-id-token")
                if shouldReturn404 {
                    throw APIError.httpError(404)
                }
                return AuthResponse(
                    token: validToken,
                    refreshToken: "refresh-01",
                    warehouseId: "wh-tashkent-01",
                    isConfigured: true
                )
            },
            loginPinHandler: { _, _ in
                XCTFail("Unexpected PIN call in OTP mode")
                throw APIError.unauthorized
            },
            uploadPushTokenHandler: {
                uploadedPush = true
            }
        )

        await vm.sendOtp()
        XCTAssertEqual(vm.error, "Phone number is required.")
        XCTAssertFalse(vm.otpSent)

        vm.phone = "  +998901234567  "
        await vm.sendOtp()
        XCTAssertNil(vm.error)
        XCTAssertTrue(vm.otpSent)
        XCTAssertEqual(sentPhone, "+998901234567")

        vm.otpCode = "123456789"
        vm.clampOtpCode()
        XCTAssertEqual(vm.otpCode, "123456")

        let store = TokenStore.shared
        store.clear()

        await vm.verifyOtp(tokenStore: store)
        XCTAssertEqual(vm.error, "No account found. Register your warehouse on the web portal.")
        XCTAssertFalse(store.isAuthenticated)

        shouldReturn404 = false
        await vm.verifyOtp(tokenStore: store)
        XCTAssertNil(vm.error)
        XCTAssertTrue(store.isAuthenticated)
        XCTAssertEqual(store.warehouseId, "wh-tashkent-01")
        XCTAssertTrue(uploadedPush)
    }

    func testLoginViewModelPinFlowAndModeToggle() async throws {
        var resetCalled = false
        let validToken = try makeJwt(homeNodeId: "wh-samarkand-02", isConfigured: true)

        let vm = LoginViewModel(
            configureAuthHandler: {},
            resetFlowHandler: { resetCalled = true },
            sendOtpHandler: { _ in },
            verifySmsHandler: { _ in "" },
            loginIdTokenHandler: { _ in throw APIError.unauthorized },
            loginPinHandler: { phone, pin in
                XCTAssertEqual(phone, "+998909998877")
                XCTAssertEqual(pin, "654321")
                return AuthResponse(
                    token: validToken,
                    refreshToken: "refresh-pin",
                    warehouseId: "wh-samarkand-02",
                    isConfigured: true
                )
            },
            uploadPushTokenHandler: {}
        )

        vm.otpSent = true
        vm.otpCode = "111222"
        vm.toggleAuthMode()
        XCTAssertFalse(vm.useOtp)
        XCTAssertFalse(vm.otpSent)
        XCTAssertEqual(vm.otpCode, "")
        XCTAssertTrue(resetCalled)

        vm.phone = "+998909998877"
        XCTAssertTrue(vm.isSubmitDisabled)

        vm.pin = "65432199"
        vm.clampPin()
        XCTAssertEqual(vm.pin, "654321")
        XCTAssertFalse(vm.isSubmitDisabled)

        let store = TokenStore.shared
        store.clear()
        await vm.loginWithPin(tokenStore: store)
        XCTAssertNil(vm.error)
        XCTAssertEqual(store.warehouseId, "wh-samarkand-02")
    }

    // MARK: - LocationSetupViewModel

    func testLocationSetupViewModelNewWarehouseValidationAndSetup() async throws {
        let store = TokenStore.shared
        store.clear()
        let unassignedJwt = try makeJwt(homeNodeId: nil, isConfigured: false)
        let configuredJwt = try makeJwt(homeNodeId: "wh-new-01", isConfigured: true)
        store.store(auth: AuthResponse(token: unassignedJwt, refreshToken: "ref-init", warehouseId: "", isConfigured: false))
        XCTAssertFalse(store.hasAssignedWarehouse)

        var canResolve = false
        let vm = LocationSetupViewModel(
            fetchLocationHandler: {
                XCTFail("Should not fetch existing location when unassigned")
                throw APIError.invalidResponse
            },
            resolveLocationHandler: { val in
                guard canResolve else { return nil }
                return AddressLocationValue(
                    address: val.address.isEmpty ? "Sergeli Zone, Tashkent" : val.address,
                    lat: 41.2234,
                    lng: 69.2178,
                    placeId: "place-sergeli"
                )
            },
            patchLocationHandler: { _, _, _, _ in
                XCTFail("Should not patch when unassigned")
                throw APIError.invalidResponse
            },
            refreshTokenHandler: { _ in
                throw APIError.invalidResponse
            },
            setupHandler: { name, address, placeId, lat, lng in
                XCTAssertEqual(name, "South Hub")
                XCTAssertEqual(address, "Sergeli Zone, Tashkent")
                XCTAssertEqual(placeId, "place-sergeli")
                XCTAssertEqual(lat, 41.2234, accuracy: 0.0001)
                XCTAssertEqual(lng, 69.2178, accuracy: 0.0001)
                let raw = try XCTUnwrap("""
                {
                  "warehouse_id": "wh-new-01",
                  "token": "\(configuredJwt)",
                  "refresh_token": "ref-updated",
                  "is_configured": true
                }
                """.data(using: .utf8))
                return try JSONDecoder().decode(WarehouseSetupResponse.self, from: raw)
            }
        )

        await vm.loadExistingIfNeeded(hasAssignedWarehouse: false)
        XCTAssertEqual(vm.warehouseName, "")

        await vm.submit(tokenStore: store)
        XCTAssertEqual(vm.error, "Select an address from the suggestions or share your location.")

        canResolve = true
        vm.warehouseName = "ab"
        await vm.submit(tokenStore: store)
        XCTAssertEqual(vm.error, "Warehouse name is required.")

        vm.warehouseName = "  South Hub  "
        await vm.submit(tokenStore: store)
        XCTAssertNil(vm.error)
        XCTAssertEqual(store.warehouseId, "wh-new-01")
        XCTAssertTrue(store.isConfigured)
    }

    func testLocationSetupViewModelAssignedWarehouseLoadAndPatch() async throws {
        let store = TokenStore.shared
        store.clear()
        let assignedUnconfiguredJwt = try makeJwt(homeNodeId: "wh-assigned-9", isConfigured: false)
        let assignedConfiguredJwt = try makeJwt(homeNodeId: "wh-assigned-9", isConfigured: true)
        store.store(auth: AuthResponse(token: assignedUnconfiguredJwt, refreshToken: "ref-9", warehouseId: "wh-assigned-9", isConfigured: false))
        XCTAssertTrue(store.hasAssignedWarehouse)

        var patchedAddress: String?
        let vm = LocationSetupViewModel(
            fetchLocationHandler: {
                let raw = try XCTUnwrap("""
                {
                  "warehouse_id": "wh-assigned-9",
                  "name": "Chilonzor Depot",
                  "address": "Chilonzor 9, Tashkent",
                  "place_id": "p-chil",
                  "lat": 41.2721,
                  "lng": 69.2045,
                  "country_code": "UZ",
                  "currency_code": "UZS"
                }
                """.data(using: .utf8))
                return try JSONDecoder().decode(WarehouseLocationResponse.self, from: raw)
            },
            resolveLocationHandler: { val in val },
            patchLocationHandler: { address, _, _, _ in
                patchedAddress = address
                let raw = try XCTUnwrap("""
                {
                  "warehouse_id": "wh-assigned-9",
                  "name": "Chilonzor Depot",
                  "address": "\(address)",
                  "lat": 41.2721,
                  "lng": 69.2045,
                  "country_code": "UZ",
                  "currency_code": "UZS"
                }
                """.data(using: .utf8))
                return try JSONDecoder().decode(WarehouseLocationResponse.self, from: raw)
            },
            refreshTokenHandler: { refresh in
                XCTAssertEqual(refresh, "ref-9")
                return AuthResponse(
                    token: assignedConfiguredJwt,
                    refreshToken: "ref-10",
                    warehouseId: "wh-assigned-9",
                    isConfigured: true
                )
            },
            setupHandler: { _, _, _, _, _ in
                throw APIError.invalidResponse
            }
        )

        await vm.loadExistingIfNeeded(hasAssignedWarehouse: true)
        XCTAssertEqual(vm.warehouseName, "Chilonzor Depot")
        XCTAssertEqual(vm.packCountry, "UZ")
        XCTAssertEqual(vm.location.address, "Chilonzor 9, Tashkent")

        await vm.submit(tokenStore: store)
        XCTAssertNil(vm.error)
        XCTAssertEqual(patchedAddress, "Chilonzor 9, Tashkent")
        XCTAssertTrue(store.isConfigured)
    }

    // MARK: - AddressLocationFieldViewModel

    func testAddressLocationFieldSearchPickAndFallback() async {
        let vm = AddressLocationFieldViewModel(
            autocompleteHandler: { query in
                [GeocodePrediction(placeId: "p-1", description: "\(query), Tashkent")]
            },
            resolvePlaceHandler: { placeId in
                if placeId == "fail" {
                    throw APIError.httpError(502)
                }
                return ResolvedLocation(address: "Yunusobod 4, Tashkent", lat: 41.3456, lng: 69.2891, placeId: placeId)
            },
            reverseHandler: { lat, lng in
                ResolvedLocation(address: "Tashkent Center", lat: lat, lng: lng, placeId: "rev-1")
            },
            resolveLocationValueHandler: { val in
                AddressLocationValue(address: val.address, lat: 41.3000, lng: 69.2400, placeId: "fallback-id")
            },
            coordinateProvider: { nil }
        )

        await vm.search("ab")
        XCTAssertTrue(vm.suggestions.isEmpty)

        await vm.search("Yunus")
        XCTAssertEqual(vm.suggestions.count, 1)
        XCTAssertEqual(vm.suggestions.first?.placeId, "p-1")

        let picked = await vm.pick(
            placeId: "p-1",
            fallback: "Yunus, Tashkent",
            currentValue: AddressLocationValue()
        )
        XCTAssertEqual(picked.address, "Yunusobod 4, Tashkent")
        XCTAssertEqual(picked.placeId, "p-1")
        XCTAssertTrue(vm.suggestions.isEmpty)

        let fallbackPicked = await vm.pick(
            placeId: "fail",
            fallback: "Fallback Street",
            currentValue: AddressLocationValue()
        )
        XCTAssertEqual(fallbackPicked.address, "Fallback Street")
        XCTAssertEqual(fallbackPicked.placeId, "fallback-id")

        let gpsResolved = await vm.useMyLocation(currentValue: AddressLocationValue())
        XCTAssertEqual(gpsResolved.address, "Tashkent Center")
        XCTAssertEqual(gpsResolved.lat, 41.2995, accuracy: 0.0001)
        XCTAssertEqual(gpsResolved.lng, 69.2401, accuracy: 0.0001)
    }

    // MARK: - CashierViewModel Async Workflows

    func testCashierViewModelStartSessionAnd409DuplicateSeal() async {
        var shouldConflict = true
        let vm = CashierViewModel(
            startSessionHandler: { req in
                if shouldConflict {
                    throw APIError.httpError(409)
                }
                return CashierSessionView(
                    sessionId: "sess-101",
                    supplierId: req.supplierId,
                    depotId: req.depotId,
                    driverId: req.driverId,
                    routeId: req.routeId,
                    bagSealBarcode: req.bagSealBarcode,
                    attemptNumber: 1,
                    countedCashMinor: 0,
                    status: "BLIND_COUNT_PENDING",
                    createdAt: "2026-11-12T10:00:00Z"
                )
            }
        )

        await vm.startSession()
        XCTAssertNil(vm.session)
        XCTAssertEqual(vm.errorMessage, "Bag Seal Conflict (409): Duplicate deposit bag seal barcode already logged.")

        shouldConflict = false
        await vm.startSession()
        XCTAssertNil(vm.errorMessage)
        XCTAssertEqual(vm.session?.sessionId, "sess-101")
        XCTAssertTrue(vm.isBlindCountMasked)
        XCTAssertFalse(vm.isRecountRequired)
    }

    func testCashierViewModelBlindCount409RecountKeepsExpectedMaskedThenBalances() async {
        var callCount = 0
        let vm = CashierViewModel(
            startSessionHandler: { req in
                CashierSessionView(
                    sessionId: "sess-202",
                    supplierId: req.supplierId,
                    depotId: req.depotId,
                    driverId: req.driverId,
                    routeId: req.routeId,
                    bagSealBarcode: req.bagSealBarcode,
                    attemptNumber: 1,
                    countedCashMinor: 0,
                    status: "BLIND_COUNT_PENDING",
                    createdAt: "2026-11-12T10:00:00Z"
                )
            },
            submitBlindCountHandler: { sessionId, req in
                XCTAssertEqual(sessionId, "sess-202")
                callCount += 1
                if callCount == 1 {
                    throw APIError.httpError(409)
                }
                let totalMinor = req.denominationsUzs.reduce(Int64(0)) { acc, item in
                    acc + (Int64(item.key) ?? 0) * item.value * 100
                }
                return CashierSessionView(
                    sessionId: sessionId,
                    supplierId: "sup-1",
                    depotId: "depot-1",
                    driverId: "drv-1",
                    routeId: "route-1",
                    bagSealBarcode: "BAG-202",
                    attemptNumber: 2,
                    countedCashMinor: totalMinor,
                    expectedCashMinor: totalMinor,
                    varianceMinor: 0,
                    status: "BALANCED",
                    clearanceTokenHash: "CLR-SHA256-OK",
                    createdAt: "2026-11-12T10:00:00Z"
                )
            }
        )

        await vm.submitBlindCount()
        XCTAssertEqual(vm.errorMessage, "Please start or select a session first")

        await vm.startSession()
        vm.quickFillSampleDiscrepant()
        await vm.submitBlindCount()
        XCTAssertTrue(vm.isRecountRequired)
        XCTAssertTrue(vm.isBlindCountMasked, "Expected cash MUST remain masked during blind recount #2")
        XCTAssertEqual(vm.attemptNumber, 2)

        vm.quickFillSampleBalanced()
        await vm.submitBlindCount()
        XCTAssertTrue(vm.isBalanced)
        XCTAssertFalse(vm.isRecountRequired)
        XCTAssertFalse(vm.isBlindCountMasked)
        XCTAssertEqual(vm.clearanceTokenHash, "CLR-SHA256-OK")
    }

    func testCashierViewModelRRNReclassificationCITHandoverAndAML409() async {
        var shouldExceedAML = false
        let vm = CashierViewModel(
            startSessionHandler: { req in
                CashierSessionView(
                    sessionId: "sess-303",
                    supplierId: req.supplierId,
                    depotId: req.depotId,
                    driverId: req.driverId,
                    routeId: req.routeId,
                    bagSealBarcode: req.bagSealBarcode,
                    attemptNumber: 2,
                    countedCashMinor: 120_000_000,
                    status: "DISCREPANT_QUARANTINE",
                    createdAt: "2026-11-12T10:00:00Z"
                )
            },
            reclassifyTenderHandler: { req in
                XCTAssertEqual(req.gatewayRrn, "998877665544")
                XCTAssertEqual(req.disputedCashMinor, 22_500_000)
                return CashierSessionView(
                    sessionId: req.sessionId,
                    supplierId: "sup-1",
                    depotId: "depot-1",
                    driverId: "drv-1",
                    routeId: "route-1",
                    bagSealBarcode: "BAG-303",
                    attemptNumber: 2,
                    countedCashMinor: 120_000_000,
                    expectedCashMinor: 120_000_000,
                    varianceMinor: 0,
                    status: "BALANCED",
                    clearanceTokenHash: "CLR-RRN-FIXED",
                    createdAt: "2026-11-12T10:00:00Z"
                )
            },
            executeCITHandoverHandler: { req in
                DepotVaultLedgerEntry(
                    vaultEntryId: "vault-01",
                    depotId: req.depotId,
                    supplierId: req.supplierId,
                    sessionId: req.sessionIds.first,
                    citManifestId: "CIT-MNF-77",
                    entryType: "CIT_HANDOVER",
                    amountMinor: req.cashMinor,
                    runningBalanceMinor: 0,
                    courierBadgeId: req.courierBadgeId,
                    sealBarcode: req.citSealBarcode,
                    createdAt: "2026-11-12T11:00:00Z"
                )
            },
            checkAMLHandler: { req in
                if shouldExceedAML {
                    throw APIError.httpError(409)
                }
                return AMLCheckResponse(
                    legalEntityTin: req.legalEntityTin,
                    accumulatedCashMinor: req.cashMinor,
                    ceilingMinor: 0
                )
            }
        )

        await vm.startSession()
        await vm.reclassifyTender(orderId: "ord-77", gatewayRrn: "998877665544", disputedCashMinor: 22_500_000)
        XCTAssertTrue(vm.isBalanced)
        XCTAssertEqual(vm.clearanceTokenHash, "CLR-RRN-FIXED")

        await vm.executeCITHandover(courierBadgeId: "INKAS-01", citSealBarcode: "CIT-01", amountMinor: 142_500_000)
        XCTAssertEqual(vm.lastVaultEntry?.citManifestId, "CIT-MNF-77")

        await vm.checkAMLExposure(tin: "123456789", cashMinor: 500_000_000)
        XCTAssertFalse(vm.amlExceeded)
        XCTAssertEqual(vm.amlCeilingMinor, 0)

        shouldExceedAML = true
        await vm.checkAMLExposure(tin: "123456789", cashMinor: 9_900_000_000)
        XCTAssertTrue(vm.amlExceeded)
        XCTAssertNotNil(vm.errorMessage)
    }

    // MARK: - YardGateViewModel

    func testYardGateViewModelSealQuarantineOutboundHardBlockAndSubmission() async {
        var networkCalls = 0
        let vm = YardGateViewModel(
            recordGateEventHandler: { evt in
                networkCalls += 1
                return YardGateCheckEvent(
                    eventId: "gate-evt-1",
                    depotId: evt.depotId,
                    supplierId: evt.supplierId,
                    vehicleId: evt.vehicleId,
                    licensePlate: evt.licensePlate,
                    direction: evt.direction,
                    vehiclePurpose: evt.vehiclePurpose,
                    boltSealBarcode: evt.boltSealBarcode,
                    sealIntact: evt.sealIntact,
                    assignedBaySlot: evt.assignedBaySlot,
                    checkedAt: "2026-11-12T12:00:00Z"
                )
            }
        )

        if let secondPreset = vm.vehiclePresets.dropFirst().first {
            vm.applyPreset(secondPreset)
            XCTAssertEqual(vm.vehicleId, "vh-man-04")
            XCTAssertEqual(vm.licensePlate, "01 882 BBB")
        }

        vm.updateSealIntegrity(false)
        XCTAssertFalse(vm.sealIntact)
        XCTAssertEqual(vm.assignedBaySlot, "SECURITY_QUARANTINE_BAY")

        vm.direction = "GATE_OUT"
        XCTAssertTrue(vm.isOutboundGateBlocked)
        await vm.submitGateEvent()
        XCTAssertEqual(networkCalls, 0, "Tampered outbound seal must be hard-blocked before network dispatch")
        XCTAssertEqual(vm.errorMessage, "Outbound gate exit is hard-blocked when bolt seal is tampered.")

        vm.updateSealIntegrity(true)
        XCTAssertFalse(vm.isOutboundGateBlocked)
        XCTAssertEqual(vm.assignedBaySlot, "BAY_3_INSPECTION")
        await vm.submitGateEvent()
        XCTAssertEqual(networkCalls, 1)
        XCTAssertNil(vm.errorMessage)
        XCTAssertEqual(vm.recordedEvent?.eventId, "gate-evt-1")
    }

    // MARK: - BarcodeScannerViewModel

    func testBarcodeScannerViewModelManualAndOpticalScanLifecycle() async throws {
        let vm = BarcodeScannerViewModel(resetDelay: .milliseconds(30), playSounds: false)
        XCTAssertEqual(vm.scanState, .idle)
        XCTAssertTrue(vm.isManualSubmitDisabled)

        vm.handleBarcodeReceived("   ")
        XCTAssertEqual(vm.scanState, .error("Empty barcode"))

        vm.manualBarcode = "  SKU-UZ-001  "
        XCTAssertFalse(vm.isManualSubmitDisabled)
        XCTAssertTrue(vm.submitManualBarcode())
        XCTAssertEqual(vm.manualBarcode, "")
        XCTAssertEqual(vm.scanState, .success("SKU-UZ-001"))
        XCTAssertEqual(vm.scannedRecords.map(\.code), ["SKU-UZ-001"])

        vm.handleBarcodeReceived("BIN-A12")
        XCTAssertEqual(vm.scannedRecords.map(\.code), ["BIN-A12", "SKU-UZ-001"])

        try await Task.sleep(for: .milliseconds(80))
        XCTAssertEqual(vm.scanState, .idle)

        vm.deleteRecords(at: IndexSet(integer: 0))
        XCTAssertEqual(vm.scannedRecords.map(\.code), ["SKU-UZ-001"])

        vm.clearHistory()
        XCTAssertTrue(vm.scannedRecords.isEmpty)
        XCTAssertEqual(vm.scanState, .idle)
    }
}
