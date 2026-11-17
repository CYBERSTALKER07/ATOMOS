import Foundation

enum SupplierPortalLinks {
    /// Dev default: supplier-portal on port 3000. Override with `PEGASUS_PORTAL_HOST` (host:port, no scheme).
    static var baseURL: URL {
        let raw = ProcessInfo.processInfo.environment["PEGASUS_PORTAL_HOST"] ?? "localhost:3000"
        if raw.hasPrefix("http://") || raw.hasPrefix("https://") {
            let trimmed = raw.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
            return SupplierURL.parse(trimmed) ?? SupplierURL.portalDefault
        }
        return SupplierURL.parse("http://\(raw)") ?? SupplierURL.portalDefault
    }

    static func url(for feature: SupplierPortalFeature) -> URL {
        guard var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false) else {
            return baseURL
        }
        let trimmed = feature.portalPath.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
        components.path = trimmed.isEmpty ? "/" : "/\(trimmed)"
        return components.url ?? baseURL
    }
}
