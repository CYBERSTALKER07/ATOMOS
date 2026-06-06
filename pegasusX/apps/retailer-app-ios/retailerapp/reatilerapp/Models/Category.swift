import Foundation

// MARK: - Product Category

struct ProductCategory: Codable, Identifiable, Hashable {
    let id: String
    let name: String
    let icon: String
    let productCount: Int?
    let supplierCount: Int?

    enum CodingKeys: String, CodingKey {
        case id, name, icon
        case productCount = "product_count"
        case productCountCamel = "productCount"
        case supplierCount = "supplier_count"
        case supplierCountCamel = "supplierCount"
    }

    init(id: String, name: String, icon: String, productCount: Int?, supplierCount: Int? = nil) {
        self.id = id
        self.name = name
        self.icon = icon
        self.productCount = productCount
        self.supplierCount = supplierCount
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        name = try container.decode(String.self, forKey: .name)
        icon = (try? container.decode(String.self, forKey: .icon)) ?? "square.grid.2x2.fill"
        productCount = try container.decodeIfPresent(Int.self, forKey: .productCount)
            ?? container.decodeIfPresent(Int.self, forKey: .productCountCamel)
        supplierCount = try container.decodeIfPresent(Int.self, forKey: .supplierCount)
            ?? container.decodeIfPresent(Int.self, forKey: .supplierCountCamel)
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .id)
        try container.encode(name, forKey: .name)
        try container.encode(icon, forKey: .icon)
        try container.encodeIfPresent(productCount, forKey: .productCount)
        try container.encodeIfPresent(supplierCount, forKey: .supplierCount)
    }
}

extension ProductCategory {
    static let samples: [ProductCategory] = [
        ProductCategory(id: "cat-001", name: "Drinks", icon: "cup.and.saucer.fill", productCount: 24),
        ProductCategory(id: "cat-002", name: "Snacks", icon: "birthday.cake.fill", productCount: 18),
        ProductCategory(id: "cat-003", name: "Fruits", icon: "leaf.fill", productCount: 15),
        ProductCategory(id: "cat-004", name: "Water", icon: "drop.fill", productCount: 12),
        ProductCategory(id: "cat-005", name: "Tea", icon: "mug.fill", productCount: 8),
        ProductCategory(id: "cat-006", name: "Dairy", icon: "cup.and.saucer.fill", productCount: 20),
        ProductCategory(id: "cat-007", name: "Bakery", icon: "storefront.fill", productCount: 14),
        ProductCategory(id: "cat-008", name: "Produce", icon: "carrot.fill", productCount: 22),
    ]
}

// MARK: - Auto-Order Settings

struct SimpleAutoOrderSettings: Codable {
    var globalEnabled: Bool
    var supplierSettings: [String: Bool]  // supplier_id -> enabled
    var productSettings: [String: Bool]   // product_id -> enabled

    enum CodingKeys: String, CodingKey {
        case globalEnabled = "global_auto_order_enabled"
        case supplierSettings = "supplier_settings"
        case productSettings = "product_settings"
    }

    static let `default` = SimpleAutoOrderSettings(
        globalEnabled: false,
        supplierSettings: [:],
        productSettings: [:]
    )
}
