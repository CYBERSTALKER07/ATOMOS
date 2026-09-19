import Foundation

/// GS-R / GS-C5 leftover: pin API base to session `home_cell`.
enum CellApi {
    static let cellAPIURLs: [String: String] = [
        "cell-uz": "https://api.pegasusx.app",
        "cell-eu": "https://api-eu.pegasusx.app",
        "cell-us": "https://api-us.pegasusx.app",
        "cell-kz": "https://api-kz.pegasusx.app",
    ]

    static func trimApiBase(_ url: String) -> String {
        url.trimmingCharacters(in: .whitespacesAndNewlines).trimmingCharacters(in: CharacterSet(charactersIn: "/"))
    }

    static func isDevApiBootstrap(_ url: String) -> Bool {
        let raw = trimApiBase(url).lowercased()
        if raw.isEmpty { return true }
        if raw == "/api" || raw.hasSuffix("/api") { return true }
        let withScheme = raw.contains("://") ? raw : "http://\(raw)"
        guard let host = URL(string: withScheme)?.host?.lowercased() else { return true }
        if host == "localhost" || host == "127.0.0.1" || host == "10.0.2.2" { return true }
        if host.hasPrefix("192.168.") || host.hasPrefix("10.") { return true }
        if host.hasPrefix("172.") {
            let second = Int(host.split(separator: ".").dropFirst().first ?? "0") ?? 0
            if (16...31).contains(second) { return true }
        }
        return false
    }

    static func homeCellFromJwt(_ token: String?) -> String {
        guard let token, token.split(separator: ".").count >= 2 else { return "" }
        let payload = token.split(separator: ".")[1]
        var b64 = payload.replacingOccurrences(of: "-", with: "+").replacingOccurrences(of: "_", with: "/")
        let pad = (4 - b64.count % 4) % 4
        b64 += String(repeating: "=", count: pad)
        guard let data = Data(base64Encoded: b64),
              let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any]
        else { return "" }
        return (json["home_cell"] as? String ?? "").lowercased().trimmingCharacters(in: .whitespacesAndNewlines)
    }

    static func pinApiBaseUrl(bootstrap: String, homeCell: String? = nil, sessionApiUrl: String? = nil) -> String {
        let boot = trimApiBase(bootstrap)
        if isDevApiBootstrap(boot) { return boot.isEmpty ? "http://localhost:8180" : boot }
        let fromSession = trimApiBase(sessionApiUrl ?? "")
        if !fromSession.isEmpty { return fromSession }
        let cell = (homeCell ?? "").lowercased().trimmingCharacters(in: .whitespacesAndNewlines)
        return cellAPIURLs[cell] ?? boot
    }
}
