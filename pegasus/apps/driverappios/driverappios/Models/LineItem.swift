//
//  LineItem.swift
//  driverappios
//

import Foundation

enum LineItemStatus: String, Codable {
    case DELIVERED
    case REJECTED_DAMAGED
}

struct LineItem: Codable, Identifiable {
    let line_item_id: String
    let sku_id: String
    let quantity: Int
    let unit_price: Int
    var status: LineItemStatus

    var id: String { line_item_id }

    var lineTotal: Int { quantity * unit_price }
}

// MARK: - Mock Data

extension LineItem {
    static let mockLineItems: [LineItem] = [
        LineItem(
            line_item_id: "LI-001",
            sku_id: "SKU-COKE-500",
            quantity: 4,
            unit_price: 11_980,
            status: .DELIVERED
        ),
        LineItem(
            line_item_id: "LI-002",
            sku_id: "SKU-FANTA-15L",
            quantity: 2,
            unit_price: 23_970,
            status: .DELIVERED
        ),
        LineItem(
            line_item_id: "LI-003",
            sku_id: "SKU-SPRITE-2L",
            quantity: 2,
            unit_price: 19_980,
            status: .DELIVERED
        ),
    ]
}
