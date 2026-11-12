import Foundation
import Observation

struct WarehouseClientPolicyResponse: Decodable, Sendable {
    let outdated: Bool
    let forceUpdate: Bool
    let updateDeferred: Bool?
    let minimumVersion: String
    let recommendedVersion: String?
    let updateURL: String?
    let deferReason: String?

    enum CodingKeys: String, CodingKey {
        case outdated
        case forceUpdate = "force_update"
        case updateDeferred = "update_deferred"
        case minimumVersion = "minimum_version"
        case recommendedVersion = "recommended_version"
        case updateURL = "update_url"
        case deferReason = "defer_reason"
    }
}

@Observable
@MainActor
final class RootViewModel {
    var clientPolicyMessage: String?
    var clientPolicyForce = false
    var pendingManifest: AutoUpdater.Manifest?

    private let realtime = WarehouseRealtimeClient()
    private let fetchPolicyHandler: @Sendable (String) async throws -> WarehouseClientPolicyResponse
    private let evaluatePolicyHandler: @Sendable (WarehouseClientPolicyResponse) async -> AutoUpdater.UpdateState

    init(
        fetchPolicyHandler: @escaping @Sendable (String) async throws -> WarehouseClientPolicyResponse = { version in
            try await APIClient.shared.get(
                "v1/platform/client-policy",
                query: [
                    "role": EnterpriseUpdateConfig.policyRole,
                    "platform": "ios",
                    "version": version,
                    "channel": EnterpriseUpdateConfig.channel,
                ]
            )
        },
        evaluatePolicyHandler: @escaping @Sendable (WarehouseClientPolicyResponse) async -> AutoUpdater.UpdateState = { policy in
            await AutoUpdater.shared.evaluate(
                outdated: policy.outdated,
                forceUpdate: policy.forceUpdate,
                updateDeferred: policy.updateDeferred ?? false,
                minimumVersion: policy.minimumVersion,
                recommendedVersion: policy.recommendedVersion,
                deferReason: policy.deferReason,
                updateURL: policy.updateURL
            )
        }
    ) {
        self.fetchPolicyHandler = fetchPolicyHandler
        self.evaluatePolicyHandler = evaluatePolicyHandler
    }

    func loadClientPolicy(versionOverride: String? = nil) async {
        let version = versionOverride
            ?? (Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String)
            ?? "1.0.0"
        do {
            let policy = try await fetchPolicyHandler(version)
            let state = await evaluatePolicyHandler(policy)
            clientPolicyMessage = state.message
            clientPolicyForce = state.force
            pendingManifest = state.manifest
            if state.force, state.available {
                AutoUpdater.shared.promptInstall(manifest: state.manifest, force: true)
            }
        } catch let policyError {
            if !clientPolicyForce {
                pendingManifest = nil
            }
            _ = policyError.localizedDescription
        }
    }

    func dismissBanner() {
        guard !clientPolicyForce else { return }
        clientPolicyMessage = nil
    }

    func promptInstall() {
        AutoUpdater.shared.promptInstall(manifest: pendingManifest, force: clientPolicyForce)
    }

    func connectRealtime(realtimeHub: WarehouseRealtimeHub) {
        realtime.connect(
            onStateChange: { _ in },
            onEvent: { event in
                guard !event.type.hasPrefix("SYSTEM") else { return }
                realtimeHub.bump()
            },
            onReconnect: {
                realtimeHub.bumpReconnect()
            }
        )
    }

    func disconnectRealtime() {
        realtime.disconnect()
    }
}
