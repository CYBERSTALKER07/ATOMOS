import Foundation

/// Probes dispatch and manifests after the transport returns.
/// Screens reload from `SupplierRealtimeHub.reconnectEpoch`. This probe only
/// reports a failure; it does not pretend a discarded response refreshed them.
enum SupplierSessionReconcile {
    static func notice(previewError: String?, manifestsError: String?) -> String? {
        previewError ?? manifestsError
    }

    static func run() async {
        async let previewError = failure { try await SupplierOperationsService.dispatchPreview() }
        async let manifestsError = failure { try await SupplierOperationsService.manifests() }
        let message = notice(previewError: await previewError, manifestsError: await manifestsError)
        guard let message else { return }
        await MainActor.run {
            SupplierStatusCenter.shared.notice = message
        }
    }

    private static func failure<T>(_ work: () async throws -> T) async -> String? {
        do {
            _ = try await work()
            return nil
        } catch {
            return error.localizedDescription
        }
    }
}
