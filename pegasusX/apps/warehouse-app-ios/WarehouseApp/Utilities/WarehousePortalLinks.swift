import Foundation

enum WarehousePortalLinks {
    private static let fallbackURL = URL(string: "http://localhost:3002") ?? URL(fileURLWithPath: "/")

    /// Dev default: warehouse-portal on port 3002. Override with `PEGASUS_PORTAL_HOST` (host:port, no scheme).
    static var baseURL: URL {
        let raw = ProcessInfo.processInfo.environment["PEGASUS_PORTAL_HOST"] ?? "localhost:3002"
        let trimmed = raw.trimmingCharacters(in: CharacterSet(charactersIn: "/").union(.whitespacesAndNewlines))
        if trimmed.hasPrefix("http://") || trimmed.hasPrefix("https://") {
            return URL(string: trimmed) ?? fallbackURL
        }
        return URL(string: "http://\(trimmed)") ?? fallbackURL
    }

    static func url(for feature: WarehousePortalFeature) -> URL {
        guard var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false) else {
            return baseURL
        }
        let trimmed = feature.portalPath.trimmingCharacters(in: CharacterSet(charactersIn: "/").union(.whitespacesAndNewlines))
        components.path = trimmed.isEmpty ? "/" : "/\(trimmed)"
        return components.url ?? baseURL
    }
}
