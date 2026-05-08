import Foundation
import Observation

@MainActor
@Observable
final class RetailerRefreshCenter {
    static let shared = RetailerRefreshCenter()

    private(set) var refreshToken = 0
    private(set) var lastRefreshAt = Date.distantPast

    private init() {}

    func trigger() {
        refreshToken &+= 1
        lastRefreshAt = Date()
    }
}
