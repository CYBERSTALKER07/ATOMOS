import re

with open("apps/payload-app-ios/payload-app-ios/Models/Models.swift", "r") as f:
    content = f.read()

pattern1 = re.compile(r'struct SealOrderRequest: Encodable \{\n\s*let orderId: String\n\s*let terminalId: String\n\s*let manifestCleared: Bool\n\s*enum CodingKeys: String, CodingKey \{\n\s*case orderId = "order_id"\n\s*case terminalId = "terminal_id"\n\s*case manifestCleared = "manifest_cleared"\n\s*\}')
replacement1 = r"""struct SealOrderRequest: Encodable {
    let manifestId: String
    let orderId: String
    let terminalId: String
    let manifestCleared: Bool
    enum CodingKeys: String, CodingKey {
        case manifestId = "manifest_id"
        case orderId = "order_id"
        case terminalId = "terminal_id"
        case manifestCleared = "manifest_cleared"
    }"""
content = pattern1.sub(replacement1, content)

with open("apps/payload-app-ios/payload-app-ios/Models/Models.swift", "w") as f:
    f.write(content)

with open("apps/payload-app-ios/payload-app-ios/Services/APIClient.swift", "r") as f:
    content2 = f.read()

pattern2 = re.compile(r'func sealOrder\(orderId: String, terminalId: String\) async throws -> SealOrderResponse \{\n\s*try await post\(\n\s*"v1/payload/seal",\n\s*body: SealOrderRequest\(orderId: orderId, terminalId: terminalId, manifestCleared: true\),')
replacement2 = r"""func sealOrder(manifestId: String, orderId: String, terminalId: String) async throws -> SealOrderResponse {
        try await post(
            "v1/payload/seal",
            body: SealOrderRequest(manifestId: manifestId, orderId: orderId, terminalId: terminalId, manifestCleared: true),"""
content2 = pattern2.sub(replacement2, content2)

with open("apps/payload-app-ios/payload-app-ios/Services/APIClient.swift", "w") as f:
    f.write(content2)
