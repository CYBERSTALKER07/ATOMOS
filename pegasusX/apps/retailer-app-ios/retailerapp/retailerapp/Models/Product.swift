import Foundation

// MARK: - Strict Int64 Money Formatting

enum RetailerMoney {
    /// Formats tiyin (1/100 major unit) into `major.minor` string using pure Int64 arithmetic.
    static func formatTiyin(_ minor: Int64) -> String {
        let sign = minor < 0 ? "-" : ""
        let magnitude = minor == Int64.min ? Int64.max : abs(minor)
        let major = magnitude / 100
        let remainder = magnitude % 100
        return String(format: "%@%lld.%02lld", sign, major, remainder)
    }

    /// Decodes a JSON integer or decimal literal into Int64 without floating-point intermediates.
    static func decodeInt64<K: CodingKey>(from container: KeyedDecodingContainer<K>, forKey key: K) -> Int64? {
        if let intVal = try? container.decodeIfPresent(Int64.self, forKey: key) {
            return intVal
        }
        if var decVal = try? container.decodeIfPresent(Decimal.self, forKey: key) {
            var rounded = Decimal()
            NSDecimalRound(&rounded, &decVal, 0, .plain)
            return NSDecimalNumber(decimal: rounded).int64Value
        }
        return nil
    }
}

// MARK: - Variant

struct Variant: Codable, Identifiable, Hashable {
    let id: String
    let size: String
    let pack: String
    let packCount: Int
    let weightPerUnit: String
    /// Unit price in minor units (whole UZS / tiyin as delivered by the backend). 100% Int64 arithmetic.
    let priceMinor: Int64

    enum CodingKeys: String, CodingKey {
        case id, size, pack
        case price
        case priceMinorKey = "price_minor"
        case packCount = "pack_count"
        case packCountCamel = "packCount"
        case weightPerUnit = "weight_per_unit"
        case weightPerUnitCamel = "weightPerUnit"
    }

    init(id: String, size: String, pack: String, packCount: Int, weightPerUnit: String, priceMinor: Int64) {
        self.id = id
        self.size = size
        self.pack = pack
        self.packCount = packCount
        self.weightPerUnit = weightPerUnit
        self.priceMinor = priceMinor
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        size = try container.decodeIfPresent(String.self, forKey: .size) ?? "Standard"
        pack = try container.decodeIfPresent(String.self, forKey: .pack) ?? "Per unit"
        packCount = try container.decodeIfPresent(Int.self, forKey: .packCount)
            ?? container.decodeIfPresent(Int.self, forKey: .packCountCamel)
            ?? 1
        priceMinor = RetailerMoney.decodeInt64(from: container, forKey: .priceMinorKey)
            ?? RetailerMoney.decodeInt64(from: container, forKey: .price)
            ?? 0

        // weightPerUnit can arrive as String or numeric from backend
        if let s = try? container.decodeIfPresent(String.self, forKey: .weightPerUnit) {
            weightPerUnit = s
        } else if let s = try? container.decodeIfPresent(String.self, forKey: .weightPerUnitCamel) {
            weightPerUnit = s
        } else if let dec = try? container.decodeIfPresent(Decimal.self, forKey: .weightPerUnit) {
            weightPerUnit = NSDecimalNumber(decimal: dec).stringValue
        } else if let dec = try? container.decodeIfPresent(Decimal.self, forKey: .weightPerUnitCamel) {
            weightPerUnit = NSDecimalNumber(decimal: dec).stringValue
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
        try container.encode(priceMinor, forKey: .price)
    }
}

// MARK: - Product Offer

struct ProductOffer: Codable, Hashable {
    let productID: String
    let listPriceMinor: Int64
    let salePriceMinor: Int64?
    let discountBps: Int64?
    let promotionID: String?
    let promotionName: String?
    let promotionLabel: String?
    let promotionEndsAt: String?

    enum CodingKeys: String, CodingKey {
        case productID = "product_id"
        case listPriceMinor = "list_price_minor"
        case salePriceMinor = "sale_price_minor"
        case discountBps = "discount_bps"
        case promotionID = "promotion_id"
        case promotionName = "promotion_name"
        case promotionLabel = "promotion_label"
        case promotionEndsAt = "promotion_ends_at"
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
    let priceMinor: Int64?
    let availableStock: Int?
    let isOutOfStockFlag: Bool
    let acceptsBackorder: Bool
    let showStockCounts: Bool
    let maxQuantity: Int?
    let offer: ProductOffer?

    var isOutOfStock: Bool {
        (availableStock.map { $0 <= 0 } ?? false) || isOutOfStockFlag
    }
    var isLowStock: Bool { availableStock.map { (1...5).contains($0) } ?? false }
    var blocksAddToCart: Bool { isOutOfStock && !acceptsBackorder }
    var cartMaxQuantity: Int? {
        if let maxQuantity, maxQuantity > 0 { return maxQuantity }
        if showStockCounts, let availableStock, availableStock > 0, !acceptsBackorder { return availableStock }
        if let availableStock, availableStock > 0, !acceptsBackorder { return availableStock }
        return nil
    }

    enum CodingKeys: String, CodingKey {
        case id, name, description, nutrition, variants, offer
        case productID = "product_id"
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
        case isOutOfStockFlag = "is_out_of_stock"
        case acceptsBackorder = "accepts_backorder"
        case showStockCounts = "show_stock_counts"
        case maxQuantity = "max_quantity"
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
        priceMinor: Int64? = nil,
        availableStock: Int? = nil,
        isOutOfStockFlag: Bool = false,
        acceptsBackorder: Bool = false,
        showStockCounts: Bool = false,
        maxQuantity: Int? = nil,
        offer: ProductOffer? = nil
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
        self.priceMinor = priceMinor
        self.availableStock = availableStock
        self.isOutOfStockFlag = isOutOfStockFlag
        self.acceptsBackorder = acceptsBackorder
        self.showStockCounts = showStockCounts
        self.maxQuantity = maxQuantity
        self.offer = offer
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decodeIfPresent(String.self, forKey: .id)
            ?? container.decode(String.self, forKey: .productID)
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
        priceMinor = RetailerMoney.decodeInt64(from: container, forKey: .price)
        availableStock = try container.decodeIfPresent(Int.self, forKey: .availableStock)
            ?? container.decodeIfPresent(Int.self, forKey: .availableStockCamel)
        isOutOfStockFlag = try container.decodeIfPresent(Bool.self, forKey: .isOutOfStockFlag) ?? false
        acceptsBackorder = try container.decodeIfPresent(Bool.self, forKey: .acceptsBackorder) ?? false
        showStockCounts = try container.decodeIfPresent(Bool.self, forKey: .showStockCounts) ?? false
        maxQuantity = try container.decodeIfPresent(Int.self, forKey: .maxQuantity)
        offer = try container.decodeIfPresent(ProductOffer.self, forKey: .offer)
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
        try container.encodeIfPresent(priceMinor, forKey: .price)
        try container.encodeIfPresent(availableStock, forKey: .availableStock)
    }

    var defaultVariant: Variant? { variants.first }
    var hasSaleOffer: Bool {
        guard let sale = offer?.salePriceMinor, sale > 0 else { return false }
        return true
    }

    var displayListPrice: String? {
        if let list = offer?.listPriceMinor, list > 0 {
            return "\(list.formatted())"
        }
        if let priceMinor {
            return "\(priceMinor.formatted())"
        }
        return nil
    }

    var displayPrice: String {
        if let sale = offer?.salePriceMinor, sale > 0 {
            return "\(sale.formatted())"
        }
        if let v = defaultVariant {
            return "\(v.priceMinor.formatted())"
        }
        if let priceMinor {
            return "\(priceMinor.formatted())"
        }
        return "—"
    }

    var promotionLabel: String? {
        offer?.promotionLabel
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
