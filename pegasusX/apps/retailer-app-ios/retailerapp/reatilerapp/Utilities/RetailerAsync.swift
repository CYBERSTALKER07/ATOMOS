import Foundation

enum RetailerAsync {
    static func run(_ operation: @escaping @Sendable () async -> Void) {
        Task { await operation() }
    }
}
