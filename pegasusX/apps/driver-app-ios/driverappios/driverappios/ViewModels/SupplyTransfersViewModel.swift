//
//  SupplyTransfersViewModel.swift
//  driverappios
//

import CoreLocation
import Foundation
import SwiftUI
import Observation

@Observable
@MainActor
final class SupplyTransfersViewModel {
    var transfers: [SupplyTransferRow] = []
    var isLoading = false
    var isArriving: String?
    var errorMessage: String?
    var successMessage: String?

    private let arriveableStates: Set<String> = ["IN_TRANSIT", "IN_TRANSIT_TO_WAREHOUSE", "DISPATCHED"]

    var activeCount: Int {
        transfers.filter { $0.state.uppercased() != "ARRIVED" }.count
    }

    func refresh() async {
        isLoading = true
        errorMessage = nil
        defer { isLoading = false }
        do {
            let response = try await APIClient.shared.getSupplyTransfers()
            transfers = response.transfers
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func canArrive(_ transfer: SupplyTransferRow) -> Bool {
        arriveableStates.contains(transfer.state.uppercased())
    }

    func markArrived(_ transfer: SupplyTransferRow) async {
        isArriving = transfer.transferId
        errorMessage = nil
        successMessage = nil
        defer { isArriving = nil }

        let coordinate = await currentCoordinate()
        do {
            let response = try await APIClient.shared.arriveSupplyTransfer(
                transferId: transfer.transferId,
                latitude: coordinate.latitude,
                longitude: coordinate.longitude
            )
            successMessage = "Transfer \(response.transferId.suffix(6)) marked arrived"
            await refresh()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func currentCoordinate() async -> CLLocationCoordinate2D {
        await withCheckedContinuation { continuation in
            let manager = CLLocationManager()
            manager.requestWhenInUseAuthorization()
            if let location = manager.location {
                continuation.resume(returning: location.coordinate)
            } else {
                continuation.resume(returning: CLLocationCoordinate2D(latitude: 0, longitude: 0))
            }
        }
    }
}
