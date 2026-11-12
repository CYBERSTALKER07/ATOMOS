import XCTest
@testable import FactoryAppIOS

@MainActor
final class FactoryViewModelsTests: XCTestCase {

    private func decodeJSON<T: Decodable>(_ json: String) throws -> T {
        let data = Data(json.utf8)
        return try JSONDecoder().decode(T.self, from: data)
    }

    // MARK: - LoginViewModel

    func testLoginViewModelOTPSanitizationAndModeSwitching() {
        let vm = LoginViewModel(
            configureAuth: {},
            resetAuthFlow: {}
        )
        XCTAssertEqual(vm.mode, .otp)
        vm.phone = "   "
        XCTAssertFalse(vm.canSendOtp)

        vm.phone = "+998901234567"
        XCTAssertTrue(vm.canSendOtp)

        vm.sanitizeOtp("12a34b56789")
        XCTAssertEqual(vm.otpCode, "123456")
        XCTAssertTrue(vm.canVerifyOtp)

        vm.switchMode()
        XCTAssertEqual(vm.mode, .passwordDev)
        XCTAssertFalse(vm.otpSent)
        XCTAssertEqual(vm.otpCode, "")
        XCTAssertFalse(vm.canPasswordLogin)

        vm.password = "Secret123!"
        XCTAssertTrue(vm.canPasswordLogin)
    }

    func testLoginViewModelSendAndVerifyOTPFlow() async throws {
        let mockAuth: AuthResponse = try decodeJSON("""
        {"token":"jwt-token-1","refresh_token":"refresh-token-1","factory_id":"fac-01","factory_name":"Tashkent Dairy"}
        """)
        var authenticatedResponse: AuthResponse?
        let vm = LoginViewModel(
            configureAuth: {},
            resetAuthFlow: {},
            sendVerification: { phone in
                XCTAssertEqual(phone, "+998901234567")
            },
            verifyCode: { code in
                XCTAssertEqual(code, "654321")
                return "firebase-id-token-xyz"
            },
            loginWithIdToken: { idToken in
                XCTAssertEqual(idToken, "firebase-id-token-xyz")
                return mockAuth
            }
        )

        vm.phone = "+998901234567"
        await vm.sendOtp()
        XCTAssertTrue(vm.otpSent)
        XCTAssertNil(vm.error)

        vm.sanitizeOtp("654321")
        XCTAssertTrue(vm.canVerifyOtp)

        await vm.verifyOtp { response in
            authenticatedResponse = response
        }
        XCTAssertEqual(authenticatedResponse?.token, "jwt-token-1")
        XCTAssertEqual(authenticatedResponse?.factoryId, "fac-01")
    }

    func testLoginViewModelPasswordLoginValidationAndSuccess() async throws {
        let mockAuth: AuthResponse = try decodeJSON("""
        {"token":"jwt-pass","refresh_token":"rt-pass","factory_id":"fac-02","factory_name":"Samarkand Plant"}
        """)
        var capturedAuth: AuthResponse?
        let vm = LoginViewModel(
            configureAuth: {},
            resetAuthFlow: {},
            loginWithPassword: { phone, password in
                XCTAssertEqual(phone, "+998901112233")
                XCTAssertEqual(password, "Pass1234")
                return mockAuth
            }
        )

        vm.switchMode()
        vm.phone = "+998901112233"
        vm.password = ""
        XCTAssertFalse(vm.canPasswordLogin)
        await vm.passwordLogin { capturedAuth = $0 }
        XCTAssertNil(capturedAuth)

        vm.password = "Pass1234"
        XCTAssertTrue(vm.canPasswordLogin)
        await vm.passwordLogin { capturedAuth = $0 }
        XCTAssertNil(vm.error)
        XCTAssertEqual(capturedAuth?.factoryName, "Samarkand Plant")
    }

    // MARK: - LocationSetupViewModel

    func testLocationSetupViewModelValidationAndSubmit() async throws {
        let mockSetup: FactorySetupResponse = try decodeJSON("""
        {"factory_id":"fac-bukhara","token":"new-jwt","refresh_token":"new-rt","is_configured":true}
        """)
        var updatedToken: String?
        var updatedFactoryId: String?
        let vm = LocationSetupViewModel(
            setupFactory: { name, address, placeId, lat, lng in
                XCTAssertEqual(name, "Bukhara Hub")
                XCTAssertEqual(address, "12 Navoi Ave")
                XCTAssertEqual(placeId, "pid-99")
                XCTAssertEqual(lat, 39.7747, accuracy: 0.0001)
                XCTAssertEqual(lng, 64.4286, accuracy: 0.0001)
                return mockSetup
            },
            resolveLocation: { loc in
                loc.lat != 0 ? loc : nil
            }
        )

        vm.factoryName = "AB"
        vm.location = AddressLocationValue(address: "12 Navoi Ave", lat: 39.7747, lng: 64.4286, placeId: "pid-99")
        await vm.submit(hasAssignedFactory: false, currentRefreshToken: "old-rt") { token, _, facId in
            updatedToken = token
            updatedFactoryId = facId
        }
        XCTAssertEqual(vm.error, "Factory name is required.")
        XCTAssertNil(updatedToken)

        vm.factoryName = "Bukhara Hub"
        await vm.submit(hasAssignedFactory: false, currentRefreshToken: "old-rt") { token, _, facId in
            updatedToken = token
            updatedFactoryId = facId
        }
        XCTAssertNil(vm.error)
        XCTAssertEqual(updatedToken, "new-jwt")
        XCTAssertEqual(updatedFactoryId, "fac-bukhara")
    }

    // MARK: - CreateTransferViewModel & LoadingBayViewModel

    func testCreateTransferViewModelValidationAndCreate() async throws {
        let mockDriver: FactoryFleetDriverRow = try decodeJSON("""
        {"driver_id":"drv-1","name":"Aziz","on_shift":true}
        """)
        let mockVehicle: FactoryFleetVehicleRow = try decodeJSON("""
        {"vehicle_id":"veh-1","plate_no":"01A777AA","state":"AVAILABLE"}
        """)
        let mockResp: FactoryCreateTransferResponse = try decodeJSON("""
        {"transfer_id":"tr-900","state":"CREATED","total_vu":45}
        """)

        var createdRequest: FactoryCreateTransferRequest?
        let vm = CreateTransferViewModel(
            fetchDrivers: { [mockDriver] },
            fetchVehicles: { [mockVehicle] },
            createAction: { req in
                createdRequest = req
                return mockResp
            }
        )

        await vm.loadFleet()
        XCTAssertEqual(vm.vehicles.count, 1)
        XCTAssertEqual(vm.drivers.count, 1)

        vm.totalVu = "0"
        let failed = await vm.submit()
        XCTAssertNil(failed)
        XCTAssertEqual(vm.error, "Total VU must be a positive number.")

        vm.totalVu = "45"
        vm.orderId = "  ord-123  "
        vm.vehicleId = "veh-1"
        vm.driverId = "drv-1"
        let createdId = await vm.submit()
        XCTAssertEqual(createdId, "tr-900")
        XCTAssertEqual(createdRequest?.orderId, "ord-123")
        XCTAssertEqual(createdRequest?.vehicleId, "veh-1")
        XCTAssertEqual(createdRequest?.driverId, "drv-1")
        XCTAssertEqual(createdRequest?.totalVu, 45)
    }

    func testLoadingBayViewModelPartitioningAndBatchDispatch() async throws {
        let transfers: [Transfer] = try decodeJSON("""
        [
          {"id":"t-app-1","state":"APPROVED","total_items":10,"total_volume_l":50,"items":[]},
          {"id":"t-load-1","state":"LOADING","total_items":20,"total_volume_l":100,"items":[]},
          {"id":"t-disp-1","state":"DISPATCHED","total_items":15,"total_volume_l":75,"items":[]}
        ]
        """)
        let pulseEvents: [FactoryPulseEvent] = try decodeJSON("""
        [
          {"id":"ev-1","kind":"MANIFEST_DISPATCHED","title":"Bay 2 handoff","occurred_at":"2026-11-12T10:00:00Z"},
          {"id":"ev-2","kind":"OTHER_EVENT","title":"Ignored","occurred_at":"2026-11-12T09:00:00Z"}
        ]
        """)
        let dispatchResp: DispatchResponse = try decodeJSON("""
        {"status":"OK","manifest_id":"man-500","truck_plate":"01B555BB","stop_count":1,"created_manifest_count":1}
        """)

        var dispatchedIds: [String] = []
        let vm = LoadingBayViewModel(
            fetchBayTransfers: { transfers },
            fetchPulse: { pulseEvents },
            dispatchTransfers: { ids in
                dispatchedIds = ids
                return dispatchResp
            }
        )

        await vm.load()
        XCTAssertEqual(vm.approved.map(\.id), ["t-app-1"])
        XCTAssertEqual(vm.loadingState.map(\.id), ["t-load-1"])
        XCTAssertEqual(vm.dispatched.map(\.id), ["t-disp-1"])
        XCTAssertEqual(vm.handoffEvents.count, 1)
        XCTAssertNil(vm.handoffError)

        await vm.batchDispatch()
        XCTAssertEqual(dispatchedIds, ["t-load-1"])
        XCTAssertNil(vm.error)
    }

    // MARK: - SupplyRequestsViewModel

    func testSupplyRequestsViewModelFilteringQCAndFulfillModal() async throws {
        let req1: SupplyRequest = try decodeJSON("""
        {
          "request_id":"sr-1",
          "warehouse_id":"wh-alpha123",
          "factory_id":"fac-1",
          "supplier_id":"sup-1",
          "state":"READY",
          "priority":"URGENT",
          "requested_delivery_date":"2026-11-15T00:00:00Z",
          "total_volume_vu":120,
          "notes":"Handle with care",
          "transfer_order_id":"to-1",
          "created_by":"usr-1",
          "created_at":"2026-11-12T08:00:00Z",
          "sla_status":"AT_RISK",
          "sla_hours_remaining":4
        }
        """)
        let qcPass: SupplyRequestQCResponse = try decodeJSON("""
        {"request_id":"sr-1","result":"PASS"}
        """)
        let qcFail: SupplyRequestQCResponse = try decodeJSON("""
        {"request_id":"sr-1","result":"FAIL"}
        """)
        let fulfillOpts: SupplyFulfillOptions = try decodeJSON("""
        {
          "transfer_mode":"FACTORY_TRUCK",
          "warehouse_id":"wh-alpha123",
          "warehouse_name":"Alpha Warehouse",
          "co_located":false,
          "outcome_internal":"Internal fleet ready",
          "outcome_truck":"Truck assigned",
          "linked_driver_eta":"18m"
        }
        """)
        let transResp: SupplyRequestTransitionResponse = try decodeJSON("""
        {"request_id":"sr-1","state":"FULFILLED"}
        """)

        var transitionedAction: String?
        let vm = SupplyRequestsViewModel(
            fetchRequests: { [req1] },
            fetchQC: { _ in qcPass },
            postQC: { _, _ in qcFail },
            fetchFulfillOptions: { _ in fulfillOpts },
            transitionAction: { id, action in
                transitionedAction = "\(id):\(action)"
                return transResp
            }
        )

        await vm.load()
        XCTAssertEqual(vm.requests.count, 1)
        XCTAssertEqual(vm.qcById["sr-1"], "PASS")
        XCTAssertEqual(vm.runtimeTone, FactoryRuntimeTone.live)

        vm.selectedFilter = "SUBMITTED"
        XCTAssertTrue(vm.filteredRequests.isEmpty)
        vm.selectedFilter = "READY"
        XCTAssertEqual(vm.filteredRequests.count, 1)

        await vm.handleQC(request: req1, result: "FAIL")
        XCTAssertEqual(vm.qcById["sr-1"], "FAIL")

        await vm.handleAction(request: req1, action: "FULFILL")
        let modalItem: SupplyFulfillModalItem = try XCTUnwrap(vm.fulfillModalItem)
        XCTAssertEqual(modalItem.options.warehouseName, "Alpha Warehouse")
        XCTAssertEqual(modalItem.options.linkedDriverETA, "18m")

        await vm.confirmFulfill(modalItem)
        XCTAssertNil(vm.fulfillModalItem)
        XCTAssertEqual(transitionedAction, "sr-1:FULFILL")
    }

    // MARK: - StaffViewModel & StaffDetailViewModel

    func testStaffViewModelCreateAndDetailPinValidation() async throws {
        let member1: StaffMember = try decodeJSON("""
        {"id":"st-1","name":"Dilshod","phone":"+998901112233","role":"FACTORY_OPERATOR","status":"ACTIVE","joined_at":"2026-01-01"}
        """)
        let member2: StaffMember = try decodeJSON("""
        {"id":"st-2","name":"Kamola","role":"FACTORY_OPERATOR","status":"ACTIVE"}
        """)
        var createdStaff: (String, String)?
        let staffVM = StaffViewModel(
            fetchStaff: { [member1] },
            createStaffAction: { name, role in
                createdStaff = (name, role)
                return member2
            }
        )

        await staffVM.load()
        XCTAssertEqual(staffVM.staff.count, 1)

        staffVM.presentCreate()
        XCTAssertFalse(staffVM.canSubmitCreate)
        let emptyResult = await staffVM.createStaff()
        XCTAssertFalse(emptyResult)
        XCTAssertEqual(staffVM.createError, "Staff name is required.")

        staffVM.newStaffName = "Kamola"
        staffVM.newStaffRole = "   "
        let createSuccess = await staffVM.createStaff()
        XCTAssertTrue(createSuccess)
        XCTAssertEqual(createdStaff?.0, "Kamola")
        XCTAssertEqual(createdStaff?.1, "FACTORY_OPERATOR")

        var updatedPin: String?
        let detailVM = StaffDetailViewModel(
            staffId: "st-1",
            fetchDetail: { _ in member1 },
            setPasswordAction: { _, pin in updatedPin = pin }
        )
        await detailVM.load()
        XCTAssertEqual(detailVM.staff?.name, "Dilshod")

        detailVM.pin = "123"
        XCTAssertFalse(detailVM.canSetPassword)
        await detailVM.setPassword()
        XCTAssertNil(updatedPin)

        detailVM.pin = "4829"
        XCTAssertTrue(detailVM.canSetPassword)
        await detailVM.setPassword()
        XCTAssertEqual(updatedPin, "4829")
        XCTAssertEqual(detailVM.setMsg, "Password set")
        XCTAssertEqual(detailVM.pin, "")
    }

    // MARK: - PayloadOverrideViewModel & EnterpriseUpdateConfig

    func testPayloadOverrideViewModelAndEnterpriseUpdateURL() async throws {
        let manifestA: Manifest = try decodeJSON("""
        {"manifest_id":"m-a","truck_plate":"01A001AA","driver_name":"Driver A","state":"LOADING","transfers":[{"transfer_id":"tr-10","product_name":"Dairy Crate","quantity":10,"volume_vu":30,"state":"LOADING"}]}
        """)
        let manifestB: Manifest = try decodeJSON("""
        {"manifest_id":"m-b","truck_plate":"01B002BB","driver_name":"Driver B","state":"LOADING","transfers":[]}
        """)
        let rebalanceResp: ManifestRebalanceResponse = try decodeJSON("""
        {"source_manifest_id":"m-a","target_manifest_id":"m-b","transfers_moved":1,"volume_moved_vu":30,"reason":"OVERRIDE"}
        """)
        let cancelTransferResp: ManifestCancelTransferResponse = try decodeJSON("""
        {"manifest_id":"m-a","transfer_id":"tr-10","status":"CANCELLED"}
        """)
        let cancelManifestResp: ManifestCancelResponse = try decodeJSON("""
        {"manifest_id":"m-a","status":"CANCELLED","transfers_released":1}
        """)

        var rebalancedPair: (String, String, String)?
        let vm = PayloadOverrideViewModel(
            fetchLoadingManifests: { [manifestA, manifestB] },
            rebalanceAction: { src, dst, tid in
                rebalancedPair = (src, dst, tid)
                return rebalanceResp
            },
            cancelTransferAction: { _, _ in cancelTransferResp },
            cancelManifestAction: { _ in cancelManifestResp }
        )

        await vm.load()
        XCTAssertEqual(vm.manifests.count, 2)

        let transferToMove = try XCTUnwrap(manifestA.transfers.first)
        vm.promptMove(manifestId: "m-a", transfer: transferToMove)
        let candidate = try XCTUnwrap(vm.moveCandidate)
        XCTAssertEqual(candidate.sourceManifestId, "m-a")

        await vm.moveTransfer(candidate, targetManifestId: "m-b")
        XCTAssertEqual(rebalancedPair?.0, "m-a")
        XCTAssertEqual(rebalancedPair?.1, "m-b")
        XCTAssertEqual(rebalancedPair?.2, "tr-10")
        XCTAssertNil(vm.moveCandidate)

        XCTAssertNotNil(EnterpriseUpdateConfig.defaultManifestURL)
        let wsURL = APIClient.shared.factoryWebSocketURL(token: "tok-123", lastEventId: "ev-9")
        XCTAssertTrue(wsURL.absoluteString.contains("token=tok-123"))
        XCTAssertTrue(wsURL.absoluteString.contains("last_event_id=ev-9"))
    }
}
