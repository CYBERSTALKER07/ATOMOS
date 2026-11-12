import SwiftUI

struct ProfileView: View {
    @State private var vm = ProfileViewModel()
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @AppStorage("aiAutoOrder") private var aiAutoOrder = false
    @AppStorage("globalAutoOrder") private var globalAutoOrder = false
    @AppStorage("notificationsEnabled") private var notificationsEnabled = true

    @Environment(AuthManager.self) private var auth

    private var user: User {
        auth.currentUser ?? User(id: "", name: "—", company: "—", email: "—", avatarURL: nil)
    }
    private var displayName: String { vm.profileName.isEmpty ? user.name : vm.profileName }
    private var displayCompany: String { vm.profileCompany.isEmpty ? user.company : vm.profileCompany }

    var body: some View {
        @Bindable var vm = vm
        ScrollView {
            VStack(spacing: AppTheme.spacingLG) {
                UserCard(
                    displayName: displayName,
                    displayCompany: displayCompany,
                    userEmail: user.email,
                    profilePhone: vm.profilePhone
                )
                .accessibilityElement(children: .combine)
                .accessibilityLabel("Profile: \(displayName), \(displayCompany)")
                .slideIn(delay: 0)

                StatsRowView(
                    orderCount: vm.orderCount,
                    totalSpent: vm.totalSpent,
                    totalSpentCurrency: vm.totalSpentCurrency
                )
                .slideIn(delay: 0.05)

                CreditProfileSection(
                    profile: vm.creditProfile,
                    isLoading: vm.creditLoading,
                    missing: vm.creditMissing,
                    error: vm.creditError
                )
                .slideIn(delay: 0.06)

                OrderHistoryLink(orderCount: vm.orderCount)
                    .pegasusMinimumHitTarget()
                    .slideIn(delay: 0.07)

                EmpathyEngineSection(
                    globalAutoOrder: $globalAutoOrder,
                    showHistoryAlert: $vm.showHistoryAlert,
                    toggleAction: { enabled, useHistory in
                        await vm.toggleGlobalAutoOrder(enabled: enabled, useHistory: useHistory)
                    }
                )
                .slideIn(delay: 0.09)

                if !vm.pricingRulesSummary.isEmpty {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Pricing rules")
                            .font(.system(.subheadline, design: .rounded, weight: .bold))
                        Text(vm.pricingRulesSummary)
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, AppTheme.spacingLG)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Pricing rules: \(vm.pricingRulesSummary)")
                }

                SettingsSectionView(title: "Company", icon: "building.2.fill", items: [
                    SettingsItem(icon: "building.2", title: "Company Info", subtitle: user.company, view: "AccountProfile"),
                    SettingsItem(icon: "square.stack.3d.up", title: "Store capabilities", subtitle: "Team, stock, POS packs", view: "Capabilities"),
                    SettingsItem(icon: "mappin.and.ellipse", title: "Locations", subtitle: "Branches and checkout store", view: "Locations"),
                    SettingsItem(icon: "shippingbox", title: "Store stock", subtitle: "Receive, putaway, count", view: "StoreStock"),
                    SettingsItem(icon: "tag.fill", title: "Local SKUs", subtitle: "Non-Pegasus POS goods", view: "LocalSkus"),
                    SettingsItem(icon: "cart.fill", title: "POS", subtitle: "Cashier sales and voids", view: "POS"),
                    SettingsItem(icon: "clock.fill", title: "Shifts", subtitle: "Clock in and cash recon", view: "Shifts"),
                    SettingsItem(icon: "square.grid.2x2.fill", title: "Sections", subtitle: "Departments and SKU map", view: "Sections"),
                    SettingsItem(icon: "rectangle.split.3x3.fill", title: "Planograms", subtitle: "Shelf layout & vision audits", view: "Planograms"),
                    SettingsItem(icon: "chart.bar.doc.horizontal", title: "Reports Pro", subtitle: "Sales and inventory digest", view: "ReportsPro"),
                    SettingsItem(icon: "hand.raised.fill", title: "Floor assist", subtitle: "Section help tickets", view: "Assist"),
                    SettingsItem(icon: "person.3.fill", title: "Team", subtitle: "Staff roles and invites", view: "Team"),
                    SettingsItem(icon: "creditcard", title: "Billing", subtitle: "Manage payment methods", view: "SavedCards"),
                    SettingsItem(icon: "key", title: "API Access", subtitle: "Developer settings"),
                    SettingsItem(icon: "person.2.fill", title: "Family contacts", subtitle: "Legacy name/phone list", view: "FamilyMembers"),
                ])
                .slideIn(delay: 0.1)

                PreferencesSection(
                    aiAutoOrder: $aiAutoOrder,
                    notificationsEnabled: $notificationsEnabled,
                    onAutoOrderToggle: { enabled in
                        globalAutoOrder = enabled
                        Task { await vm.toggleGlobalAutoOrder(enabled: enabled, useHistory: true) }
                    }
                )
                .slideIn(delay: 0.15)

                SettingsSectionView(title: "Support", icon: "questionmark.circle.fill", items: [
                    SettingsItem(icon: "questionmark.circle", title: "Help Center", subtitle: nil),
                    SettingsItem(icon: "envelope", title: "Contact Support", subtitle: nil),
                    SettingsItem(icon: "doc.text", title: "Terms of Service", subtitle: nil),
                ])
                .slideIn(delay: 0.2)

                Text("Pegasus Retailer v1.0.0")
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
                    .padding(.top, AppTheme.spacingMD)
                    .padding(.bottom, AppTheme.spacingXXL)
            }
            .padding(AppTheme.spacingLG)
        }
        .scrollDismissesKeyboard(.interactively)
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .task(id: refreshCenter.refreshToken) {
            await refreshAll()
        }
        .refreshable {
            await refreshAll()
        }
    }

    private func refreshAll() async {
        let retailerId = auth.currentUser?.id ?? ""
        if let syncedGlobal = await vm.loadAll(retailerId: retailerId, currentUser: auth.currentUser) {
            globalAutoOrder = syncedGlobal
        }
    }
}

#Preview {
    NavigationStack {
        ProfileView()
            .environment(AuthManager.shared)
    }
}
