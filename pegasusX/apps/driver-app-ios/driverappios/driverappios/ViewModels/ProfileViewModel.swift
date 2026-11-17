//
//  ProfileViewModel.swift
//  driverappios
//

import Foundation
import SwiftData
import Observation

@Observable
@MainActor
final class ProfileViewModel {

    var showOfflineVerifier = false
    var showSyncQueue = false
    var showEndSession = false
    var showFleetDVIR = false
    var showMidRouteRescue = false
    var showYardGate = false
    var showSettings = false
    var pendingCount = 0

    private let queue: DriverOfflineQueue

    convenience init() {
        self.init(queue: DriverOfflineQueue.shared)
    }

    init(queue: DriverOfflineQueue) {
        self.queue = queue
    }

    func attach(container: ModelContainer) {
        queue.attach(container: container)
        refreshPendingCount()
    }

    func refreshPendingCount() {
        pendingCount = queue.pendingCount()
    }
}
