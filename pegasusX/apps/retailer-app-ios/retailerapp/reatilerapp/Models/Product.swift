import Foundation

// MARK: - Variant

struct Variant: Codable, Identifiable, Hashable {
    let id: String
    let size: String
    let pack: String
    let packCount: Int
    let weightPerUnit: String
    let price: Double

    enum CodingKeys: String, CodingKey {
        case id, size, pack, price
        case packCount = "pack_count"
        case packCountCamel = "packCount"
        case weightPerUnit = "weight_per_unit"
        case weightPerUnitCamel = "weightPerUnit"
    }

    init(id: String, size: String, pack: String, packCount: Int, weightPerUnit: String, price: Double) {
        self.id = id
        self.size = size
        self.pack = pack
        self.packCount = packCount
        self.weightPerUnit = weightPerUnit
        self.price = price
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        size = try container.decodeIfPresent(String.self, forKey: .size) ?? "Standard"
        pack = try container.decodeIfPresent(String.self, forKey: .pack) ?? "Per unit"
        packCount = try container.decodeIfPresent(Int.self, forKey: .packCount)
            ?? container.decodeIfPresent(Int.self, forKey: .packCountCamel)
            ?? 1
        price = try container.decodeIfPresent(Double.self, forKey: .price) ?? 0

        // weightPerUnit can arrive as String or Double from backend
        if let s = try? container.decodeIfPresent(String.self, forKey: .weightPerUnit) {
            weightPerUnit = s
        } else if let s = try? container.decodeIfPresent(String.self, forKey: .weightPerUnitCamel) {
            weightPerUnit = s
        } else if let d = try? container.decodeIfPresent(Double.self, forKey: .weightPerUnit) {
            weightPerUnit = String(format: "%.1f", d)
        } else if let d = try? container.decodeIfPresent(Double.self, forKey: .weightPerUnitCamel) {
            weightPerUnit = String(format: "%.1f", d)
        } else {
            weightPerUnit = "1 unit"
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .id)
        try container.encode(size, forKey: .size)
        try container.encode(pack, forKey: .pack)
        try container.encode(packCount, forKey: .packCount)
        try container.encode(weightPerUnit, forKey: .weightPerUnit)
        try container.encode(price, forKey: .price)
    }
}

// MARK: - Product

struct Product: Codable, Identifiable, Hashable {
    let id: String
    let name: String
    let description: String
    let nutrition: String
    let imageURL: String?
    let variants: [Variant]
    let supplierID: String?
    let supplierName: String?
    let supplierCategory: String?
    let categoryID: String?
    let categoryName: String?
    let sellByBlock: Bool
    let unitsPerBlock: Int?
    let price: Int?
    let availableStock: Int?

    var isOutOfStock: Bool { availableStock != nil && availableStock! <= 0 }
    var isLowStock: Bool { availableStock != nil && (1...5).contains(availableStock!) }

    enum CodingKeys: String, CodingKey {
        case id, name, description, nutrition, variants
        case imageURL = "image_url"
        case imageURLCamel = "imageURL"
        case supplierID = "supplier_id"
        case supplierIDCamel = "supplierId"
        case supplierName = "supplier_name"
        case supplierNameCamel = "supplierName"
        case supplierCategory = "supplier_category"
        case categoryID = "category_id"
        case categoryIDCamel = "categoryId"
        case categoryName = "category_name"
        case categoryNameCamel = "categoryName"
        case sellByBlock = "sell_by_block"
        case sellByBlockCamel = "sellByBlock"
        case unitsPerBlock = "units_per_block"
        case unitsPerBlockCamel = "unitsPerBlock"
        case price
        case availableStock = "available_stock"
        case availableStockCamel = "availableStock"
    }

    init(
        id: String,
        name: String,
        description: String,
        nutrition: String,
        imageURL: String?,
        variants: [Variant],
        supplierID: String? = nil,
        supplierName: String? = nil,
        supplierCategory: String? = nil,
        categoryID: String? = nil,
        categoryName: String? = nil,
        sellByBlock: Bool = false,
        unitsPerBlock: Int? = nil,
        price: Int? = nil,
        availableStock: Int? = nil
    ) {
        self.id = id
        self.name = name
        self.description = description
        self.nutrition = nutrition
        self.imageURL = imageURL
        self.variants = variants
        self.supplierID = supplierID
        self.supplierName = supplierName
        self.supplierCategory = supplierCategory
        self.categoryID = categoryID
        self.categoryName = categoryName
        self.sellByBlock = sellByBlock
        self.unitsPerBlock = unitsPerBlock
        self.price = price
        self.availableStock = availableStock
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        name = try container.decode(String.self, forKey: .name)
        description = try container.decodeIfPresent(String.self, forKey: .description) ?? ""
        nutrition = try container.decodeIfPresent(String.self, forKey: .nutrition) ?? ""
        imageURL = try container.decodeIfPresent(String.self, forKey: .imageURL)
            ?? container.decodeIfPresent(String.self, forKey: .imageURLCamel)
        variants = try container.decodeIfPresent([Variant].self, forKey: .variants) ?? []
        supplierID = try container.decodeIfPresent(String.self, forKey: .supplierID)
            ?? container.decodeIfPresent(String.self, forKey: .supplierIDCamel)
        supplierName = try container.decodeIfPresent(String.self, forKey: .supplierName)
            ?? container.decodeIfPresent(String.self, forKey: .supplierNameCamel)
        supplierCategory = try container.decodeIfPresent(String.self, forKey: .supplierCategory)
        categoryID = try container.decodeIfPresent(String.self, forKey: .categoryID)
            ?? container.decodeIfPresent(String.self, forKey: .categoryIDCamel)
        categoryName = try container.decodeIfPresent(String.self, forKey: .categoryName)
            ?? container.decodeIfPresent(String.self, forKey: .categoryNameCamel)
        sellByBlock = try container.decodeIfPresent(Bool.self, forKey: .sellByBlock)
            ?? container.decodeIfPresent(Bool.self, forKey: .sellByBlockCamel)
            ?? false
        unitsPerBlock = try container.decodeIfPresent(Int.self, forKey: .unitsPerBlock)
            ?? container.decodeIfPresent(Int.self, forKey: .unitsPerBlockCamel)
        price = try container.decodeIfPresent(Int.self, forKey: .price)
        availableStock = try container.decodeIfPresent(Int.self, forKey: .availableStock)
            ?? container.decodeIfPresent(Int.self, forKey: .availableStockCamel)
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .id)
        try container.encode(name, forKey: .name)
        try container.encode(description, forKey: .description)
        try container.encode(nutrition, forKey: .nutrition)
        try container.encodeIfPresent(imageURL, forKey: .imageURL)
        try container.encode(variants, forKey: .variants)
        try container.encodeIfPresent(supplierID, forKey: .supplierID)
        try container.encodeIfPresent(supplierName, forKey: .supplierName)
        try container.encodeIfPresent(supplierCategory, forKey: .supplierCategory)
        try container.encodeIfPresent(categoryID, forKey: .categoryID)
        try container.encodeIfPresent(categoryName, forKey: .categoryName)
        try container.encode(sellByBlock, forKey: .sellByBlock)
        try container.encodeIfPresent(unitsPerBlock, forKey: .unitsPerBlock)
        try container.encodeIfPresent(price, forKey: .price)
        try container.encodeIfPresent(availableStock, forKey: .availableStock)
    }

    var defaultVariant: Variant? { variants.first }
    var displayPrice: String {
        if let v = defaultVariant {
            return "\(Int(v.price).formatted())"
        }
        if let price {
            return "\(price.formatted())"
        }
        return "—"
    }

    var merchandisingLabel: String? {
        if let categoryName {
            return categoryName
        }
        if sellByBlock, let unitsPerBlock {
            return "\(unitsPerBlock) units / block"
        }
        return nil
    }
}

// MARK: - Sample Data

extension Product {
    static let samples: [Product] = [
        Product(
            id: "prod-001",
            name: "Organic Whole Milk",
            description: "Farm-fresh organic whole milk, pasteurized.",
            nutrition: "Calories: 150, Fat: 8g, Protein: 8g",
            imageURL: nil,
            variants: [
                Variant(id: "v-001a", size: "1L", pack: "Single", packCount: 1, weightPerUnit: "1.03kg", price: 3.49),
                Variant(id: "v-001b", size: "1L", pack: "6-Pack", packCount: 6, weightPerUnit: "1.03kg", price: 18.99)
            ]
        ),
        Product(
            id: "prod-002",
            name: "Sourdough Bread",
            description: "Artisan sourdough, slow-fermented 24 hours.",
            nutrition: "Calories: 120, Fat: 0.5g, Protein: 4g",
            imageURL: nil,
            variants: [
                Variant(id: "v-002a", size: "500g", pack: "Single", packCount: 1, weightPerUnit: "500g", price: 4.99),
                Variant(id: "v-002b", size: "500g", pack: "3-Pack", packCount: 3, weightPerUnit: "500g", price: 13.49)
            ]
        ),
        Product(
            id: "prod-003",
            name: "Free-Range Eggs",
            description: "Grade A free-range eggs from local farms.",
            nutrition: "Calories: 70, Fat: 5g, Protein: 6g",
            imageURL: nil,
            variants: [
                Variant(id: "v-003a", size: "12 ct", pack: "Dozen", packCount: 1, weightPerUnit: "720g", price: 5.99),
                Variant(id: "v-003b", size: "30 ct", pack: "Tray", packCount: 1, weightPerUnit: "1.8kg", price: 12.99)
            ]
        ),
        Product(
            id: "prod-004",
            name: "Avocado",
            description: "Hass avocados, ripe and ready to eat.",
            nutrition: "Calories: 240, Fat: 22g, Protein: 3g",
            imageURL: nil,
            variants: [
                Variant(id: "v-004a", size: "Single", pack: "Each", packCount: 1, weightPerUnit: "200g", price: 1.99),
                Variant(id: "v-004b", size: "Single", pack: "6-Pack", packCount: 6, weightPerUnit: "200g", price: 9.99)
            ]
        ),
        Product(
            id: "prod-005",
            name: "Sparkling Water",
            description: "Natural mineral sparkling water, unflavored.",
            nutrition: "Calories: 0, Fat: 0g, Protein: 0g",
            imageURL: nil,
            variants: [
                Variant(id: "v-005a", size: "500ml", pack: "Single", packCount: 1, weightPerUnit: "500g", price: 1.49),
                Variant(id: "v-005b", size: "500ml", pack: "12-Pack", packCount: 12, weightPerUnit: "500g", price: 14.99)
            ]
        ),
        Product(
            id: "prod-006",
            name: "Greek Yogurt",
            description: "Thick, creamy Greek yogurt with live cultures.",
            nutrition: "Calories: 100, Fat: 0g, Protein: 17g",
            imageURL: nil,
            variants: [
                Variant(id: "v-006a", size: "170g", pack: "Single", packCount: 1, weightPerUnit: "170g", price: 1.79),
                Variant(id: "v-006b", size: "170g", pack: "4-Pack", packCount: 4, weightPerUnit: "170g", price: 5.99)
            ]
        )
    ]
}
