import Foundation

enum HandoffDestination: Equatable {
    case home
    case fleetMap
    case manifestList
    case manifestDetail(String)
    case orderDetail(String)
    case unresolved
}

enum HandoffPathResolver {
    static func resolve(_ link: String) -> HandoffDestination {
        let trimmed = link.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return .unresolved }

        let path: String
        if trimmed.hasPrefix("/") {
            path = String(trimmed.split(separator: "?").first ?? Substring(trimmed))
        } else if let url = URL(string: trimmed) {
            path = url.path
        } else {
            path = trimmed
        }

        let segments = path.split(separator: "/").map(String.init).filter { !$0.isEmpty }
        guard let root = segments.first else { return .unresolved }

        switch root {
        case "manifests":
            if segments.count >= 2 { return .manifestDetail(segments[1]) }
            return .manifestList
        case "dispatch":
            return .home
        case "fleet":
            return .fleetMap
        case "orders":
            if segments.count >= 2 { return .orderDetail(segments[1]) }
            return .unresolved
        default:
            return .unresolved
        }
    }
}
