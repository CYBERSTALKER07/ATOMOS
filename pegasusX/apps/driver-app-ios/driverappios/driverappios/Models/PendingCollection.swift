//
//  PendingCollection.swift
//  driverappios
//
//  Mirror of backend-go/order/service.go::PendingCollection.
//  Returned by GET /v1/driver/pending-collections.
//

import Foundation

struct PendingCollection: Codable, Hashable, Identifiable {
    let orderId: String
    let retailerId: String
    let amount: Int64
    let state: String
    let updatedAt: String

    var id: String { orderId }

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case retailerId = "retailer_id"
        case amount
        case state
        case updatedAt = "updated_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        orderId = try c.decode(String.self, forKey: .orderId)
        retailerId = (try? c.decode(String.self, forKey: .retailerId)) ?? ""
        amount = (try? c.decode(Int64.self, forKey: .amount)) ?? 0
        state = (try? c.decode(String.self, forKey: .state)) ?? ""
        updatedAt = (try? c.decode(String.self, forKey: .updatedAt)) ?? ""
    }

    init(orderId: String, retailerId: String, amount: Int64, state: String, updatedAt: String) {
        self.orderId = orderId
        self.retailerId = retailerId
        self.amount = amount
        self.state = state
        self.updatedAt = updatedAt
    }
}
