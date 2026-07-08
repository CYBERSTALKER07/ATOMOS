//
//  DriverWsRefresh.swift
//  driverappios
//

import Foundation

enum DriverWsRefresh {
    static func shouldRefreshManifest(eventType: String) -> Bool {
        switch eventType {
        case "ORDER_ASSIGNED",
             "ORDER_REASSIGNED",
             "ORDER_STATUS_CHANGED",
             "ORDER_FINALIZED",
             "ROUTE_CREATED",
             "ROUTE_REORDERED",
             "PAYMENT_REQUIRED",
             "PAYMENT_CLEARED",
             "MANIFEST_DISPATCHED",
             "MANIFEST_COMPLETED",
             "SHOP_CLOSED_RESOLVED",
             "NEGOTIATION_RESOLVED",
             "DELIVERY_SESSION_UPDATED",
             "DRIVER_AVAILABILITY_CHANGED",
             "REASSIGN_HANDSHAKE_COMPLETED":
            return true
        default:
            return false
        }
    }
}
