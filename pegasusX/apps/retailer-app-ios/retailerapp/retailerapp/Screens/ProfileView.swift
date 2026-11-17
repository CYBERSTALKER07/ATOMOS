import SwiftUI
import UIKit

struct ProfileView: View {
    @State private var vm = ProfileViewModel()
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var preferencesStore = RetailerPreferencesStore.shared
    @State private var copiedOtp = false
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

                SettingsSectionView(title: "Company", icon: "building.2.fill", items: (
                    [
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
                    ]
                    + (vm.isOwner ? [
                        SettingsItem(icon: "person.badge.shield.checkmark", title: String(localized: "mobile_retailer.approval.inbox_title"), subtitle: "Approve or reject over-mandate orders", view: "OwnerApprovals"),
                    ] : [])
                    + [
                        SettingsItem(icon: "person.3.fill", title: "Team", subtitle: "Staff roles and invites", view: "Team"),
                        SettingsItem(icon: "creditcard", title: "Billing", subtitle: "Manage payment methods", view: "SavedCards"),
                        SettingsItem(icon: "key", title: "API Access", subtitle: "Developer settings"),
                        SettingsItem(icon: "person.2.fill", title: "Family contacts", subtitle: "Legacy name/phone list", view: "FamilyMembers"),
                    ]
                ))
                .slideIn(delay: 0.1)

                PreferencesSection(
                    aiAutoOrder: $aiAutoOrder,
                    notificationsEnabled: $notificationsEnabled,
                    preferencesStore: preferencesStore,
                    effectiveWorkspace: vm.workspaceMode,
                    onAutoOrderToggle: { enabled in
                        globalAutoOrder = enabled
                        Task { await vm.toggleGlobalAutoOrder(enabled: enabled, useHistory: true) }
                    }
                )
                .slideIn(delay: 0.15)

                telegramQuickBindCard
                    .slideIn(delay: 0.17)

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

    private var telegramQuickBindCard: some View {
        let _ = preferencesStore.currentLanguage
        return LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                SectionLabel(title: L10n.string("mobile_retailer.telegram_link.title"), icon: "paperplane.fill")

                Text(L10n.string("mobile_retailer.telegram_link.subtitle"))
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
                    .padding(.horizontal, AppTheme.spacingLG)

                if let status = vm.telegramStatus, status.linked {
                    HStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "checkmark.seal.fill")
                            .foregroundStyle(AppTheme.success)
                        Text(
                            status.username?.isEmpty == false
                                ? L10n.format("mobile_retailer.telegram_link.linked_as", status.username ?? "")
                                : L10n.string("mobile_retailer.telegram_link.linked_generic")
                        )
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.success)
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                }

                if let req = vm.telegramLinkRequest, !req.otpCode.isEmpty {
                    VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                        HStack {
                            Text(req.otpCode)
                                .font(.system(.title2, design: .monospaced, weight: .bold))
                                .foregroundStyle(AppTheme.textPrimary)
                                .textSelection(.enabled)
                                .accessibilityIdentifier("telegram-otp-code")

                            Spacer()

                            Button {
                                UIPasteboard.general.string = req.otpCode
                                copiedOtp = true
                                Haptics.selection()
                            } label: {
                                Label(
                                    copiedOtp
                                        ? L10n.string("mobile_retailer.telegram_link.copied")
                                        : L10n.string("mobile_retailer.telegram_link.copy_otp"),
                                    systemImage: copiedOtp ? "checkmark" : "doc.on.doc"
                                )
                                .font(.system(.caption, design: .rounded, weight: .semibold))
                                .padding(.horizontal, AppTheme.spacingMD)
                                .padding(.vertical, AppTheme.spacingSM)
                                .background(AppTheme.accent.opacity(0.14))
                                .foregroundStyle(AppTheme.accent)
                                .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                            }
                            .buttonStyle(.plain)
                        }

                        if let expiresAt = req.expiresAt, !expiresAt.isEmpty {
                            Text(L10n.format("mobile_retailer.telegram_link.otp_expires", req.otpCode, String(expiresAt.prefix(16))))
                                .font(.system(.caption2, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }

                        if let rawLink = req.botDeepLink ?? vm.telegramStatus?.botDeepLink,
                           let url = URL(string: rawLink) {
                            Link(destination: url) {
                                Label(L10n.string("mobile_retailer.telegram_link.open_bot"), systemImage: "arrow.up.right.square")
                                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                    .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
                                    .background(AppTheme.surfaceElevated)
                                    .foregroundStyle(AppTheme.accent)
                                    .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                            }
                        }
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                }

                if let err = vm.telegramError, !err.isEmpty {
                    Text(err)
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.destructive)
                        .padding(.horizontal, AppTheme.spacingLG)
                }

                Button {
                    copiedOtp = false
                    Task { await vm.requestTelegramOtp() }
                } label: {
                    HStack {
                        if vm.telegramLoading {
                            ProgressView()
                                .tint(.white)
                        } else {
                            Image(systemName: "key.fill")
                        }
                        Text(
                            vm.telegramLinkRequest == nil
                                ? L10n.string("mobile_retailer.telegram_link.generate_otp")
                                : L10n.string("mobile_retailer.telegram_link.refresh_otp")
                        )
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    }
                    .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
                    .background(AppTheme.accent)
                    .foregroundStyle(.white)
                    .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                }
                .disabled(vm.telegramLoading)
                .buttonStyle(.plain)
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.bottom, AppTheme.spacingMD)
            }
        }
    }
}

#Preview {
    NavigationStack {
        ProfileView()
            .environment(AuthManager.shared)
    }
}
