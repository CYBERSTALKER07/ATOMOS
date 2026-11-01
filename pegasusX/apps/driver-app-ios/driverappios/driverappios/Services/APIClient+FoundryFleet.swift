//
//  APIClient+FoundryFleet.swift
//  driverappios
//

import Foundation

extension APIClient {

    // MARK: - Foundry Fleet & Yard Operations

    func submitDVIR(inspection: FleetDVIRInspection) async throws -> FleetDVIRInspection {
        return try await post("v1/foundry/fleet/dvir", body: inspection)
    }

    func recordGateEvent(event: YardGateCheckEvent) async throws -> YardGateCheckEvent {
        return try await post("v1/foundry/fleet/gate-events", body: event)
    }

    func executeMidRouteRescue(request: MidRouteRescueRequest) async throws -> MidRouteRescueResult {
        return try await post("v1/foundry/fleet/mid-route-rescue", body: request)
    }

    func checkPredictiveReefer(request: PredictiveReeferRequest) async throws -> PredictiveReeferResponse {
        return try await post("v1/foundry/fleet/predictive-reefer", body: request)
    }

    func checkVehicleDispatch(vehicleId: String) async throws -> DispatchCheckResponse {
        return try await get("v1/foundry/fleet/vehicles/\(vehicleId)/dispatch-check")
    }
}
