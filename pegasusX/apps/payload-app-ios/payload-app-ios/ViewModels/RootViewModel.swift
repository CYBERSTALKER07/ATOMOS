//
//  RootViewModel.swift
//  payload-app-ios
//

import Foundation
import Observation

@Observable
@MainActor
final class RootViewModel {
    typealias PolicyFetcher = @Sendable (_ platform: String, _ version: String) async throws -> ClientPolicyResponse
    typealias PolicyEvaluator = @Sendable (_ policy: ClientPolicyResponse) async -> AutoUpdater.UpdateState
    typealias InstallPrompter = @MainActor (_ manifest: AutoUpdater.Manifest?, _ force: Bool) -> Void

    private(set) var clientPolicyMessage: String?
    private(set) var clientPolicyForce = false
    private(set) var pendingManifest: AutoUpdater.Manifest?

    private let fetchPolicy: PolicyFetcher
    private let evaluatePolicy: PolicyEvaluator
    private let promptInstallHandler: InstallPrompter

    init(
        fetchPolicy: @escaping PolicyFetcher = { platform, version in
            try await APIClient.shared.clientPolicy(platform: platform, version: version)
        },
        evaluatePolicy: @escaping PolicyEvaluator = { policy in
            await AutoUpdater.evaluate(
                outdated: policy.outdated,
                forceUpdate: policy.forceUpdate,
                updateDeferred: policy.updateDeferred ?? false,
                minimumVersion: policy.minimumVersion,
                recommendedVersion: policy.recommendedVersion,
                deferReason: policy.deferReason,
                updateURL: policy.updateURL
            )
        },
        promptInstall: @escaping InstallPrompter = { manifest, force in
            AutoUpdater.promptInstall(manifest: manifest, force: force)
        }
    ) {
        self.fetchPolicy = fetchPolicy
        self.evaluatePolicy = evaluatePolicy
        self.promptInstallHandler = promptInstall
    }

    func loadClientPolicy(versionOverride: String? = nil) async {
        let version = versionOverride
            ?? (Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String)
            ?? "1.0.0"
        do {
            let policy = try await fetchPolicy("ios", version)
            let state = await evaluatePolicy(policy)
            clientPolicyMessage = state.message
            clientPolicyForce = state.force
            pendingManifest = state.manifest
            if state.force, state.available {
                PegasusHaptics.warning()
                promptInstallHandler(state.manifest, true)
            }
        } catch {
            // Policy fetch is optional on local/dev stacks; preserve non-blocking shell.
            print("[RootViewModel] client policy check skipped: \(error.localizedDescription)")
        }
    }

    func dismissBanner() {
        guard !clientPolicyForce else { return }
        PegasusHaptics.light()
        clientPolicyMessage = nil
    }

    func triggerInstall() {
        PegasusHaptics.medium()
        promptInstallHandler(pendingManifest, clientPolicyForce)
    }
}
