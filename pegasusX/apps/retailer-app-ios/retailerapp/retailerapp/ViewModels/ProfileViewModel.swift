import Foundation
import Observation

@Observable
@MainActor
final class ProfileViewModel {
    var showHistoryAlert = false
    var profileName = ""
    var profileCompany = ""
    var profilePhone = ""
    var profileLocation = ""
    var pricingRulesSummary = ""
    var orderCount = 0
    var totalSpent: Int64 = 0
    var totalSpentCurrency = "UZS"
    var creditProfile: CreditProfile?
    var creditLoading = true
    var creditMissing = false
    var creditError: String?

    private let api = APIClient.shared

    private struct RetailerPricingRulesResponse: Decodable {
        let summary: String?
        let status: String?
    }

    func loadAll(retailerId: String, currentUser: User?) async -> Bool? {
        await loadProfile(currentUser: currentUser)
        let syncedGlobalAutoOrder = await loadStats(retailerId: retailerId)
        await loadCreditProfile()
        return syncedGlobalAutoOrder
    }

    func loadProfile(currentUser: User?) async {
        do {
            let profile = try await api.getProfile()
            profileName = profile.name
            profileCompany = profile.company
            profilePhone = profile.phone
            profileLocation = profile.location ?? ""
        } catch {
            profileName = currentUser?.name ?? ""
            profileCompany = currentUser?.company ?? ""
            profilePhone = ""
            profileLocation = ""
        }

        do {
            let rules: RetailerPricingRulesResponse = try await api.get(path: "/v1/retailer/pricing/rules")
            if let summary = rules.summary, !summary.isEmpty {
                pricingRulesSummary = summary
            } else if let status = rules.status, !status.isEmpty {
                pricingRulesSummary = status
            } else {
                pricingRulesSummary = "Supplier pricing rules are active for your account."
            }
        } catch {
            pricingRulesSummary = ""
        }
    }

    func loadStats(retailerId: String) async -> Bool? {
        do {
            let orders: [Order] = try await api.get(path: "/v1/retailers/\(retailerId)/orders")
            orderCount = orders.count
            totalSpent = orders.reduce(0) { $0 + $1.totalAmount }
            totalSpentCurrency = orders.first?.currency ?? "UZS"

            let settings: AutoOrderSettings = try await api.get(path: "/v1/retailer/settings/auto-order")
            return settings.globalEnabled
        } catch {
            return nil
        }
    }

    func loadCreditProfile() async {
        creditLoading = true
        creditError = nil
        creditMissing = false
        do {
            creditProfile = try await api.getCreditProfile()
        } catch let apiError as APIError {
            creditProfile = nil
            if case .serverError(let statusCode, _) = apiError, statusCode == 404 {
                creditMissing = true
            } else {
                creditError = "Credit unavailable"
            }
        } catch {
            creditProfile = nil
            creditError = "Credit unavailable"
        }
        creditLoading = false
    }

    func toggleGlobalAutoOrder(enabled: Bool, useHistory: Bool) async {
        do {
            let body: [String: Any] = ["global_auto_order_enabled": enabled, "use_history": useHistory]
            let _: [String: Bool] = try await api.patch(
                path: "/v1/retailer/settings/auto-order/global",
                body: AnyCodable(body)
            )
            Haptics.selection()
        } catch {
            Haptics.error()
        }
    }
}
