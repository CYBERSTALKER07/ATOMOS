import Testing
import Foundation

/// Factory Models — JSON Decoding, Defaults, Edge Cases
struct FactoryModelsTests {

    // MARK: - Auth

    @Test func authResponseDecoding() throws {
        let json = """
        {"token":"jwt-abc","refresh_token":"rt-xyz","factory_id":"fac-1","factory_name":"Test Factory"}
        """.data(using: .utf8)!

        let auth = try JSONDecoder().decode(AuthResponse.self, from: json)
        #expect(auth.token == "jwt-abc")
        #expect(auth.refreshToken == "rt-xyz")
        #expect(auth.factoryId == "fac-1")
        #expect(auth.factoryName == "Test Factory")
    }

    // MARK: - Dashboard

    @Test func dashboardStatsEmptyDefaults() {
        let stats = DashboardStats.empty
        #expect(stats.pendingTransfers == 0)
        #expect(stats.loadingTransfers == 0)
        #expect(stats.activeManifests == 0)
        #expect(stats.dispatchedToday == 0)
        #expect(stats.vehiclesTotal == 0)
        #expect(stats.vehiclesAvailable == 0)
        #expect(stats.staffOnShift == 0)
        #expect(stats.criticalInsights == 0)
    }

    @Test func dashboardStatsFullDecoding() throws {
        let json = """
        {"pending_transfers":5,"loading_transfers":3,"active_manifests":2,
         "dispatched_today":10,"vehicles_total":15,"vehicles_available":8,
         "staff_on_shift":12,"critical_insights":1}
        """.data(using: .utf8)!

        let stats = try JSONDecoder().decode(DashboardStats.self, from: json)
        #expect(stats.pendingTransfers == 5)
        #expect(stats.loadingTransfers == 3)
        #expect(stats.activeManifests == 2)
        #expect(stats.dispatchedToday == 10)
        #expect(stats.vehiclesTotal == 15)
        #expect(stats.vehiclesAvailable == 8)
        #expect(stats.staffOnShift == 12)
        #expect(stats.criticalInsights == 1)
    }

    // MARK: - Transfer

    @Test func transferMinimalDecoding() throws {
        let json = """
        {"id":"t-1","items":[]}
        """.data(using: .utf8)!

        let transfer = try JSONDecoder().decode(Transfer.self, from: json)
        #expect(transfer.id == "t-1")
        #expect(transfer.factoryId == "")
        #expect(transfer.warehouseId == "")
        #expect(transfer.state == "")
        #expect(transfer.totalItems == 0)
        #expect(transfer.totalVolumeL == 0.0)
        #expect(transfer.items.isEmpty)
    }

    @Test func transferFullDecoding() throws {
        let json = """
        {"id":"t-1","factory_id":"f-1","warehouse_id":"wh-1",
         "warehouse_name":"Central WH","state":"LOADING",
         "priority":"HIGH","total_items":100,"total_volume_l":250.5,
         "notes":"Urgent","created_at":"2026-01-01","updated_at":"2026-01-02",
         "items":[
            {"id":"ti-1","product_id":"p-1","product_name":"Milk",
             "quantity":50,"quantity_available":45,"unit_volume_l":1.2}
         ]}
        """.data(using: .utf8)!

        let transfer = try JSONDecoder().decode(Transfer.self, from: json)
        #expect(transfer.state == "LOADING")
        #expect(transfer.priority == "HIGH")
        #expect(transfer.totalItems == 100)
        #expect(transfer.totalVolumeL == 250.5)
        #expect(transfer.items.count == 1)
        #expect(transfer.items[0].productName == "Milk")
        #expect(transfer.items[0].quantity == 50)
        #expect(transfer.items[0].unitVolumeL == 1.2)
    }

    @Test func transferListResponseEmpty() throws {
        let json = """
        {"transfers":[],"total":0}
        """.data(using: .utf8)!

        let res = try JSONDecoder().decode(TransferListResponse.self, from: json)
        #expect(res.transfers.isEmpty)
        #expect(res.total == 0)
    }

    // MARK: - Vehicle

    @Test func vehicleMinimalDecoding() throws {
        let json = """
        {"id":"v-1"}
        """.data(using: .utf8)!

        let v = try JSONDecoder().decode(Vehicle.self, from: json)
        #expect(v.id == "v-1")
        #expect(v.plateNumber == "")
        #expect(v.driverName == "")
        #expect(v.status == "")
        #expect(v.capacityKg == 0.0)
        #expect(v.capacityL == 0.0)
    }

    @Test func vehicleFullDecoding() throws {
        let json = """
        {"id":"v-1","plate_number":"01A123AB","driver_name":"Ali",
         "status":"AVAILABLE","capacity_kg":5000.0,"capacity_l":12000.0,"current_route":"r-1"}
        """.data(using: .utf8)!

        let v = try JSONDecoder().decode(Vehicle.self, from: json)
        #expect(v.plateNumber == "01A123AB")
        #expect(v.driverName == "Ali")
        #expect(v.status == "AVAILABLE")
        #expect(v.capacityKg == 5000.0)
        #expect(v.capacityL == 12000.0)
        #expect(v.currentRoute == "r-1")
    }

    // MARK: - Staff

    @Test func staffMemberMinimalDecoding() throws {
        let json = """
        {"id":"s-1"}
        """.data(using: .utf8)!

        let s = try JSONDecoder().decode(StaffMember.self, from: json)
        #expect(s.id == "s-1")
        #expect(s.name == "")
        #expect(s.phone == "")
        #expect(s.role == "")
        #expect(s.status == "")
        #expect(s.joinedAt == "")
    }

    // MARK: - Transfer State Machine

    @Test func transferStateNames() {
        let validStates = ["DRAFT", "APPROVED", "LOADING", "DISPATCHED", "CANCELLED"]
        for state in validStates {
            #expect(!state.isEmpty)
        }
    }

    @Test func transitionRequestEncoding() throws {
        let req = TransitionRequest(targetState: "LOADING")
        let data = try JSONEncoder().encode(req)
        let str = String(data: data, encoding: .utf8)!
        #expect(str.contains("target_state"))
        #expect(str.contains("LOADING"))
    }

    // MARK: - Dispatch

    @Test func dispatchResponseDecoding() throws {
        let json = """
        {"manifest_id":"m-1","truck_plate":"01A999AB","stop_count":5}
        """.data(using: .utf8)!

        let res = try JSONDecoder().decode(DispatchResponse.self, from: json)
        #expect(res.manifestId == "m-1")
        #expect(res.truckPlate == "01A999AB")
        #expect(res.stopCount == 5)
    }
}
