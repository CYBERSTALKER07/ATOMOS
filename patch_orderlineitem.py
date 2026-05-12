import re

file_path = "pegasus/apps/retailer-app-ios/retailerapp/reatilerapp/Models/Order.swift"
with open(file_path, "r") as f:
    content = f.read()

encode_func = """
    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .legacyID)
        try container.encode(productId, forKey: .productID)
        try container.encode(productName, forKey: .productName)
        try container.encode(variantId, forKey: .variantId)
        try container.encode(variantSize, forKey: .variantSize)
        try container.encode(quantity, forKey: .quantity)
        try container.encode(unitPrice, forKey: .unitPrice)
        try container.encode(totalPrice, forKey: .totalPrice)
    }
}"""

content = re.sub(r'(\s*self\.totalPrice = decodedTotal > 0 \? decodedTotal \: \(basePrice \* Double\(self\.quantity\)\)\n\s*)\}', r'\1' + "\n" + encode_func + "\n", content)

with open(file_path, "w") as f:
    f.write(content)
