import Foundation

enum WarehousePortalLinks {
    /// Dev default: warehouse-portal on port 3002. Override with `PEGASUS_PORTAL_HOST` (host:port, no scheme).
    static var baseURL: URL {
        let raw = ProcessInfo.processInfo.environment["PEGASUS_PORTAL_HOST"] ?? "localhost:3002"
        if raw.hasPrefix("http://") || raw.hasPrefix("https://") {
            return URL(string: raw.trimmingCharacters(in: CharacterSet(charactersIn: "/")))!
        }
        return URL(string: "http://\(raw)")!
    }

    static func url(for feature: WarehousePortalFeature) -> URL {
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)!
        let trimmed = feature.portalPath.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
        components.path = trimmed.isEmpty ? "/" : "/\(trimmed)"
        return components.url ?? baseURL
    }
}
