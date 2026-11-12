import Foundation
import Observation

@Observable
@MainActor
final class SupplierShellViewModel {
    var sidebarSelection: SupplierSection? = .dashboard
    var isSidebarExpanded = true
    var compactTab: CompactTab = .dashboard
    var clientPolicyMessage: String?
    var clientPolicyForce = false
    var pendingManifest: AutoUpdater.Manifest?

    func dismissBanner() {
        clientPolicyMessage = nil
    }

    func triggerUpdateInstall() {
        if !AutoUpdater.shared.promptInstall(manifest: pendingManifest, force: clientPolicyForce) {
            SupplierStatusCenter.shared.notice = L10n.format("mobile_supplier.chrome.update_manifest_unavailable")
        }
    }

    func loadClientPolicy() async {
        let statusCenter = SupplierStatusCenter.shared
        guard let version = AppBuildVersion.marketingVersion(
            Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String
        ) else {
            let failure = AutoUpdater.failedPolicy(
                message: L10n.format("mobile_supplier.chrome.app_version_missing"),
                hadBanner: clientPolicyMessage != nil
            )
            if failure.clearBanner {
                clientPolicyMessage = nil
                clientPolicyForce = false
                pendingManifest = nil
            }
            statusCenter.notice = failure.notice
            return
        }
        do {
            struct ClientPolicy: Decodable {
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
            let policy: ClientPolicy = try await APIClient.shared.get(
                "v1/platform/client-policy",
                query: [
                    "role": EnterpriseUpdateConfig.policyRole,
                    "platform": "ios",
                    "version": version,
                    "channel": EnterpriseUpdateConfig.channel,
                ]
            )
            let state = await AutoUpdater.shared.evaluate(
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
            if state.manifestFailed {
                clientPolicyMessage = nil
                clientPolicyForce = false
                pendingManifest = nil
                statusCenter.notice = L10n.format("mobile_supplier.chrome.update_manifest_unavailable")
            } else if state.force, state.available,
               !AutoUpdater.shared.promptInstall(manifest: state.manifest, force: true) {
                statusCenter.notice = L10n.format("mobile_supplier.chrome.update_manifest_unavailable")
            }
        } catch {
            let failure = AutoUpdater.failedPolicy(
                message: error.localizedDescription,
                hadBanner: clientPolicyMessage != nil
            )
            if failure.clearBanner {
                clientPolicyMessage = nil
                clientPolicyForce = false
                pendingManifest = nil
            }
            statusCenter.notice = failure.notice
        }
    }
}

struct BillingGatewayOption: Identifiable, Equatable {
    let id: String
    var isSelected: Bool
}

@Observable
@MainActor
final class BillingGateViewModel {
    var bankName = ""
    var accountHolder = ""
    var accountNumber = ""
    var swiftBic = ""
    var iban = ""
    var gatewayOptions: [BillingGatewayOption] = [
        BillingGatewayOption(id: "GLOBAL_PAY", isSelected: false),
        BillingGatewayOption(id: "ADYEN", isSelected: false),
        BillingGatewayOption(id: "AIRWALLEX", isSelected: false),
        BillingGatewayOption(id: "CASH", isSelected: true),
    ]
    var loading = false
    var error: String?

    var selectedGateways: [String] {
        gatewayOptions.filter(\.isSelected).map(\.id).sorted()
    }

    var canSubmit: Bool {
        !bankName.isEmpty && !accountHolder.isEmpty && !accountNumber.isEmpty
            && !swiftBic.isEmpty && !selectedGateways.isEmpty
    }

    func submit(tokenStore: TokenStore) async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await SupplierService.configureBilling(
                BillingSetupRequest(
                    bankName: bankName,
                    accountHolder: accountHolder,
                    accountNumber: accountNumber,
                    swiftBic: swiftBic,
                    iban: iban.isEmpty ? nil : iban,
                    selectedGateways: selectedGateways
                )
            )
            PegasusHaptics.success()
            tokenStore.markConfigured(resp.isConfigured)
        } catch {
            PegasusHaptics.error()
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
        }
    }
}

@Observable
@MainActor
final class SupplierLoginViewModel {
    var phone = ""
    var password = ""
    var loading = false
    var error: String?

    var canLogin: Bool {
        !loading && !phone.isEmpty && !password.isEmpty
    }

    func login(tokenStore: TokenStore) async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let auth = try await SupplierService.login(phone: phone, password: password)
            guard auth.token != nil else {
                error = L10n.format("mobile_supplier.chrome.login_no_token")
                PegasusHaptics.error()
                return
            }
            PegasusHaptics.success()
            tokenStore.store(auth: auth)
            Task { await PushNotificationManager.shared.uploadStoredTokenIfPossible() }
        } catch {
            PegasusHaptics.error()
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
        }
    }
}
