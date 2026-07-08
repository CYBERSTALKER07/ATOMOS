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

