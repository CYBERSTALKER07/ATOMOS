//
//  RootViewModel.swift
//  driverappios
//

import Foundation
import Observation

struct DriverClientPolicyResponse: Decodable, Sendable {
    let outdated: Bool
    let forceUpdate: Bool
    let updateDeferred: Bool
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

    private let fetchPolicy: @Sendable (String) async throws -> DriverClientPolicyResponse

    init(
        fetchPolicy: @escaping @Sendable (String) async throws -> DriverClientPolicyResponse = { path in
            try await APIClient.shared.get(path)
        }
    ) {
        self.fetchPolicy = fetchPolicy
    }

    func dismissBanner() {
        guard !clientPolicyForce else { return }
        clientPolicyMessage = nil
    }

    func loadClientPolicy(currentVersion: String? = nil) async {
        let version = currentVersion
            ?? (Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String)
            ?? "1.0.0"
        do {
            var components = URLComponents()
            components.queryItems = [
                URLQueryItem(name: "role", value: EnterpriseUpdateConfig.policyRole),
                URLQueryItem(name: "platform", value: "ios"),
                URLQueryItem(name: "version", value: version),
                URLQueryItem(name: "channel", value: EnterpriseUpdateConfig.channel),
            ]
            let query = components.percentEncodedQuery.map { "?\($0)" } ?? ""
            let policy = try await fetchPolicy("/v1/platform/client-policy\(query)")
            let state = await AutoUpdater.evaluate(
                outdated: policy.outdated,
                forceUpdate: policy.forceUpdate,
                updateDeferred: policy.updateDeferred,
                minimumVersion: policy.minimumVersion,
                recommendedVersion: policy.recommendedVersion,
                deferReason: policy.deferReason,
                updateURL: policy.updateURL
            )
            clientPolicyMessage = state.message
            clientPolicyForce = state.force
            pendingManifest = state.manifest
            if state.force, state.available {
                AutoUpdater.promptInstall(manifest: state.manifest, force: true)
            }
        } catch {
            // Policy fetch is optional on local/dev stacks.
            print("[RootViewModel] client policy check skipped: \(error.localizedDescription)")
        }
    }
}
