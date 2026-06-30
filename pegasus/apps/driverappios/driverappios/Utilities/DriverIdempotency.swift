//
//  DriverIdempotency.swift
//  driverappios
//
//  Deterministic offline mutation keys: deliverySessionId + action + seq
//

import Foundation

enum DriverIdempotency {
    static func offlineMutation(deliverySessionId: String, action: String, seq: Int) -> String {
        let session = deliverySessionId.trimmingCharacters(in: .whitespacesAndNewlines)
        let act = action.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let safeSession = session.isEmpty ? "session-unknown" : session
        return "\(safeSession):\(act):\(seq)"
    }

    static func activeDeliverySessionId(routeId: String?) -> String {
        let route = routeId?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !route.isEmpty { return route }
        let driverId = TokenStore.shared.userId?.trimmingCharacters(in: .whitespacesAndNewlines) ?? "driver"
        let day = ISO8601DateFormatter().string(from: Date()).prefix(10)
        return "driver-\(driverId)-\(day)"
    }
}
