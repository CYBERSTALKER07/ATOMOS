import Foundation

enum JwtPayload {
    static func isConfigured(token: String?) -> Bool {
        guard let claims = decode(token) else { return false }
        return (claims["is_configured"] as? Bool) == true
    }

    static func homeNodeId(token: String?) -> String? {
        guard let claims = decode(token),
              let id = claims["home_node_id"] as? String,
              !id.isEmpty else { return nil }
        return id
    }

    static func homeCell(token: String?) -> String? {
        guard let claims = decode(token),
              let cell = (claims["home_cell"] as? String) ?? (claims["cell_id"] as? String),
              !cell.isEmpty else { return nil }
        return cell
    }

    static func supplierId(token: String?) -> String? {
        guard let claims = decode(token),
              let id = (claims["supplier_id"] as? String) ?? (claims["org_id"] as? String),
              !id.isEmpty else { return nil }
        return id
    }

    static func subject(token: String?) -> String? {
        guard let claims = decode(token),
              let sub = (claims["sub"] as? String) ?? (claims["user_id"] as? String),
              !sub.isEmpty else { return nil }
        return sub
    }

    private static func decode(_ token: String?) -> [String: Any]? {
        guard let token, !token.isEmpty else { return nil }
        let parts = token.split(separator: ".")
        guard parts.count >= 2 else { return nil }
        var base64 = String(parts[1])
            .replacingOccurrences(of: "-", with: "+")
            .replacingOccurrences(of: "_", with: "/")
        let padding = 4 - (base64.count % 4)
        if padding < 4 { base64 += String(repeating: "=", count: padding) }
        guard let data = Data(base64Encoded: base64),
              let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return nil
        }
        return json
    }
}
