import SwiftUI

struct ProfileView: View {
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @AppStorage("aiAutoOrder") private var aiAutoOrder = false
    @AppStorage("globalAutoOrder") private var globalAutoOrder = false
    @AppStorage("notificationsEnabled") private var notificationsEnabled = true
    @State private var showHistoryAlert = false
    @State private var profileName: String = ""
    @State private var profileCompany: String = ""
    @State private var profilePhone: String = ""
    @State private var profileLocation: String = ""

    @Environment(AuthManager.self) private var auth

    private var user: User { auth.currentUser ?? User(id: "", name: "—", company: "—", email: "—", avatarURL: nil) }
    private var displayName: String { profileName.isEmpty ? user.name : profileName }
    private var displayCompany: String { profileCompany.isEmpty ? user.company : profileCompany }
    private let api = APIClient.shared

    var body: some View {
        ScrollView {
            VStack(spacing: AppTheme.spacingLG) {
                // User Identity Card
                userCard.slideIn(delay: 0)

                // Stats row
                statsRow.slideIn(delay: 0.05)

                // Order History link
                orderHistoryLink.slideIn(delay: 0.07)

                // Empathy Engine — Global Auto-Order
                empathyEngineSection.slideIn(delay: 0.09)

                // Settings
                settingsSection("Company", icon: "building.2.fill", items: [
                    SettingsItem(icon: "building.2", title: "Company Info", subtitle: user.company),
                    SettingsItem(icon: "creditcard", title: "Billing", subtitle: "Manage payment methods"),
                    SettingsItem(icon: "key", title: "API Access", subtitle: "Developer settings"),
                    SettingsItem(icon: "person.2.fill", title: "Family Members", subtitle: "Manage family/staff", view: "FamilyMembers"),
                ]).slideIn(delay: 0.1)

                preferencesSection.slideIn(delay: 0.15)

                settingsSection("Support", icon: "questionmark.circle.fill", items: [
                    SettingsItem(icon: "questionmark.circle", title: "Help Center", subtitle: nil),
                    SettingsItem(icon: "envelope", title: "Contact Support", subtitle: nil),
                    SettingsItem(icon: "doc.text", title: "Terms of Service", subtitle: nil),
                ]).slideIn(delay: 0.2)

                Text("Pegasus Retailer v1.0.0")
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
                    .padding(.top, AppTheme.spacingMD)
                    .padding(.bottom, AppTheme.spacingXXL)
            }
            .padding(AppTheme.spacingLG)
        }
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .task { await loadProfile() }
        .task { await loadStats() }
        .task(id: refreshCenter.refreshToken) {
            await loadProfile()
            await loadStats()
        }
        .refreshable {
            await loadProfile()
            await loadStats()
        }
    }

    // MARK: - User Card

    private var userCard: some View {
        VStack(spacing: 0) {
            // Gradient header
            ZStack(alignment: .bottomLeading) {
                AppTheme.heroGradient
                    .frame(height: 80)

                HStack(spacing: AppTheme.spacingLG) {
                    ZStack {
                        Circle()
                            .fill(.white)
                            .frame(width: 68, height: 68)
                            .shadow(color: AppTheme.accent.opacity(0.2), radius: 8, y: 4)
                        Text(String(user.name.prefix(1)))
                            .font(.system(.title, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.accent)
                    }
                    .offset(y: 34)

                    Spacer()
                }
                .padding(.horizontal, AppTheme.spacingXL)
            }

            // Info
            VStack(alignment: .leading, spacing: AppTheme.spacingXS) {
                Text(displayName)
                    .font(.system(.title3, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)

                Text(displayCompany)
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)

                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "envelope")
                        .font(.system(size: 12))
                    Text(user.email ?? "—")
                        .font(.system(.caption, design: .rounded))
                }
                .foregroundStyle(AppTheme.textTertiary)

                if !profilePhone.isEmpty {
                    HStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "phone")
                            .font(.system(size: 12))
                        Text(profilePhone)
                            .font(.system(.caption, design: .rounded))
                    }
                    .foregroundStyle(AppTheme.textTertiary)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, AppTheme.spacingXL)
            .padding(.top, AppTheme.spacingHuge)
            .padding(.bottom, AppTheme.spacingLG)
        }
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: AppTheme.shadowRadius, x: 0, y: AppTheme.shadowOffsetY)
    }

    // MARK: - Order History Link

    private var orderHistoryLink: some View {
        NavigationLink {
            HistoryView()
        } label: {
            HStack(spacing: AppTheme.spacingMD) {
                ZStack {
                    RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                        .fill(AppTheme.surfaceElevated)
                        .frame(width: 36, height: 36)
                    Image(systemName: "clock.fill")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(AppTheme.textSecondary)
                }

                Text("Order History")
                    .font(.system(.subheadline, design: .rounded, weight: .medium))
                    .foregroundStyle(AppTheme.textPrimary)

                Spacer()

                Text("\(orderCount) orders")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)

                Image(systemName: "chevron.right")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(AppTheme.textTertiary.opacity(0.5))
            }
            .padding(AppTheme.spacingLG)
            .background(AppTheme.cardBackground)
            .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
            .shadow(color: AppTheme.shadowColor, radius: 4, y: 2)
        }
    }

    // MARK: - Empathy Engine (Global Auto-Order)

    private var empathyEngineSection: some View {
        LabCard {
            VStack(alignment: .leading, spacing: 0) {
                sectionLabel("Empathy Engine", icon: "arrow.triangle.2.circlepath")

                HStack(spacing: AppTheme.spacingMD) {
                    ZStack {
                        RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                            .fill(AppTheme.surfaceElevated)
                            .frame(width: 36, height: 36)
                        Image(systemName: "arrow.triangle.2.circlepath")
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                    }

                    VStack(alignment: .leading, spacing: 2) {
                        Text("Auto-Order Everything")
                            .font(.system(.subheadline, design: .rounded, weight: .medium))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text("Auto-order all previously ordered products")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }

                    Spacer()

                    Toggle("", isOn: Binding(
                        get: { globalAutoOrder },
                        set: { newVal in
                            globalAutoOrder = newVal
                            if newVal {
                                showHistoryAlert = true
                            } else {
                                Task { await toggleGlobalAutoOrder(enabled: false, useHistory: false) }
                            }
                        }
                    ))
                        .tint(AppTheme.accent)
                        .labelsHidden()
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.vertical, AppTheme.spacingMD)

                if globalAutoOrder {
                    HStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "checkmark.circle.fill")
                            .font(.system(size: 12))
                            .foregroundStyle(AppTheme.success)
                        Text("Global auto-order is active. This overrides individual supplier and product settings.")
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                    .padding(.bottom, AppTheme.spacingMD)
                    .transition(.move(edge: .top).combined(with: .opacity))
                }
            }
        }
        .animation(AnimationConstants.express, value: globalAutoOrder)
        .alert("Use Previous Analytics?", isPresented: $showHistoryAlert, actions: {
            Button("Use History") {
                Task { await toggleGlobalAutoOrder(enabled: true, useHistory: true) }
            }
            Button("Start Fresh") {
                Task { await toggleGlobalAutoOrder(enabled: true, useHistory: false) }
            }
            Button("Cancel", role: .cancel) {
                globalAutoOrder = false
            }
        }, message: {
            Text("Use existing order history for predictions, or start fresh? Starting fresh requires at least 2 orders per product.")
        })
    }

    // MARK: - Stats

    @State private var orderCount: Int = 0
    @State private var totalSpent: Int64 = 0
    @State private var totalSpentCurrency: String = "UZS"

    private func formatSpent(_ amount: Int64, currency: String) -> String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        formatter.groupingSeparator = " "
        formatter.maximumFractionDigits = 0
        let formatted = formatter.string(from: NSNumber(value: amount)) ?? "\(amount)"
        return "\(formatted) \(currency.isEmpty ? "UZS" : currency)"
    }

    private var statsRow: some View {
        HStack(spacing: AppTheme.spacingMD) {
            miniStat(value: "\(orderCount)", label: "Orders", icon: "shippingbox.fill", color: AppTheme.accent)
            miniStat(value: formatSpent(totalSpent, currency: totalSpentCurrency), label: "Spent", icon: "dollarsign.circle.fill", color: AppTheme.success)
            miniStat(value: "4.9", label: "Rating", icon: "star.fill", color: AppTheme.warning)
        }
    }

    private func miniStat(value: String, label: String, icon: String, color: Color) -> some View {
        VStack(spacing: AppTheme.spacingSM) {
            Image(systemName: icon)
                .font(.system(size: 18))
                .foregroundStyle(color)

            Text(value)
                .font(.system(.headline, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textPrimary)

            Text(label)
                .font(.system(.caption2, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
        .shadow(color: AppTheme.shadowColor, radius: 4, y: 2)
    }

    // MARK: - Preferences

    private var preferencesSection: some View {
        LabCard {
            VStack(alignment: .leading, spacing: 0) {
                sectionLabel("Preferences", icon: "slider.horizontal.3")

                settingsToggle(icon: "sparkles", title: "AI Auto-Order", subtitle: "Automatically place predicted orders", color: AppTheme.accent, isOn: $aiAutoOrder)

                Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight).padding(.leading, 60)

                settingsToggle(icon: "bell.fill", title: "Notifications", subtitle: "Push notification alerts", color: AppTheme.info, isOn: $notificationsEnabled)
            }
        }
    }

    // MARK: - Settings Section

    private func settingsSection(_ title: String, icon: String, items: [SettingsItem]) -> some View {
        LabCard {
            VStack(alignment: .leading, spacing: 0) {
                sectionLabel(title, icon: icon)

                ForEach(Array(items.enumerated()), id: \.element.id) { index, item in
                    settingsRow(item)

                    if index < items.count - 1 {
                        Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight).padding(.leading, 60)
                    }
                }
            }
        }
    }

    private func sectionLabel(_ title: String, icon: String) -> some View {
        HStack(spacing: AppTheme.spacingSM) {
            Image(systemName: icon)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(AppTheme.accent)
            Text(title)
                .font(.system(.caption, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textTertiary)
                .textCase(.uppercase)
        }
        .padding(.horizontal, AppTheme.spacingLG)
        .padding(.top, AppTheme.spacingMD)
        .padding(.bottom, AppTheme.spacingSM)
    }

    private func settingsRow(_ item: SettingsItem) -> some View {
        Group {
            if item.view == "FamilyMembers" {
                NavigationLink(destination: FamilyMembersView()) {
                    settingsRowContent(item)
                }
            } else {
                Button {} label: {
                    settingsRowContent(item)
                }
            }
        }
    }
    
    private func settingsRowContent(_ item: SettingsItem) -> some View {
        Button {} label: {
            HStack(spacing: AppTheme.spacingMD) {
                ZStack {
                    RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                        .fill(AppTheme.accentSoft.opacity(0.4))
                        .frame(width: 34, height: 34)
                    Image(systemName: item.icon)
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(AppTheme.accent)
                }

                VStack(alignment: .leading, spacing: 2) {
                    Text(item.title)
                        .font(.system(.subheadline, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textPrimary)
                    if let subtitle = item.subtitle {
                        Text(subtitle)
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                }

                Spacer()

                Image(systemName: "chevron.right")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(AppTheme.textTertiary.opacity(0.5))
            }
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.vertical, AppTheme.spacingMD)
        }
    }

    private func settingsToggle(icon: String, title: String, subtitle: String, color: Color, isOn: Binding<Bool>) -> some View {
        HStack(spacing: AppTheme.spacingMD) {
            ZStack {
                RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                    .fill(color.opacity(0.12))
                    .frame(width: 34, height: 34)
                Image(systemName: icon)
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(color)
            }

            VStack(alignment: .leading, spacing: 2) {
                Text(title)
                    .font(.system(.subheadline, design: .rounded, weight: .medium))
                    .foregroundStyle(AppTheme.textPrimary)
                Text(subtitle)
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            Spacer()

            Toggle("", isOn: isOn)
                .tint(AppTheme.accent)
                .labelsHidden()
        }
        .padding(.horizontal, AppTheme.spacingLG)
        .padding(.vertical, AppTheme.spacingMD)
    }

    // MARK: - API

    private func loadProfile() async {
        do {
            let profile = try await api.getProfile()
            profileName = profile.name
            profileCompany = profile.company
            profilePhone = profile.phone
            profileLocation = profile.location ?? ""
        } catch {
            print("Failed to load profile")
        }
    }

    private func loadStats() async {
        let rid = auth.currentUser?.id ?? ""
        do {
            let orders: [Order] = try await api.get(path: "/v1/retailers/\(rid)/orders")
            orderCount = orders.count
            totalSpent = orders.reduce(0) { $0 + $1.totalAmount }
            totalSpentCurrency = orders.first?.currency ?? "UZS"
            
            // Also fetch settings so toggles are perfectly in sync
            let s: AutoOrderSettings = try await api.get(path: "/v1/retailer/settings/auto-order")
            globalAutoOrder = s.globalEnabled
        } catch {}
    }

    private func toggleGlobalAutoOrder(enabled: Bool, useHistory: Bool) async {
        do {
            let body: [String: Any] = ["global_auto_order_enabled": enabled, "use_history": useHistory]
            let _: [String: Bool] = try await api.patch(
                path: "/v1/retailer/settings/auto-order/global",
                body: AnyCodable(body)
            )
        } catch {}
    }
}

private struct SettingsItem: Identifiable {
    let id = UUID()
    let icon: String
    let title: String
    let subtitle: String?
    var view: String? = nil
}

#Preview {
    NavigationStack {
        ProfileView()
            .environment(AuthManager.shared)
    }
}


// MARK: - Family Members
struct FamilyMembersView: View {
    @State private var members: [FamilyMemberResponse] = []
    @State private var isLoading = false
    @State private var showAddSheet = false
    @State private var errorMessage: String? = nil

    private let api = APIClient.shared

    var body: some View {
        List {
            if isLoading && members.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, alignment: .center)
                    .listRowBackground(Color.clear)
            } else if let errorMessage, members.isEmpty {
                VStack(spacing: AppTheme.spacingMD) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .font(.system(size: 36))
                        .foregroundStyle(.orange)
                    Text("Family Members Unavailable")
                        .font(.headline)
                    Text(errorMessage)
                        .font(.subheadline)
                        .foregroundStyle(AppTheme.textSecondary)
                        .multilineTextAlignment(.center)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 40)
                .listRowBackground(Color.clear)
            } else if members.isEmpty {
                VStack(spacing: AppTheme.spacingMD) {
                    Image(systemName: "person.2.badge.plus")
                        .font(.system(size: 40))
                        .foregroundStyle(AppTheme.textTertiary)
                    Text("No Family Members")
                        .font(.headline)
                    Text("Add family members to allow them to place orders.")
                        .font(.subheadline)
                        .foregroundStyle(AppTheme.textSecondary)
                        .multilineTextAlignment(.center)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 40)
                .listRowBackground(Color.clear)
            } else {
                ForEach(members) { member in
                    HStack(spacing: AppTheme.spacingMD) {
                        ZStack {
                            Circle().fill(AppTheme.surfaceElevated).frame(width: 40, height: 40)
                            Image(systemName: "person.fill").foregroundStyle(AppTheme.accent)
                        }
                        VStack(alignment: .leading, spacing: 2) {
                            Text(member.nickname).font(.system(.body, design: .rounded, weight: .semibold))
                            Text("Added \(member.createdAt.prefix(10))").font(.caption).foregroundStyle(AppTheme.textTertiary)
                        }
                    }
                    .padding(.vertical, 4)
                }
                .onDelete { offsets in
                    let membersToDelete = offsets.map { members[$0] }
                    for m in membersToDelete {
                        Task {
                            do {
                                try await api.removeFamilyMember(memberId: m.id)
                                errorMessage = nil
                                await loadMembers()
                            } catch {
                                errorMessage = RetailerErrorSupport.message(
                                    for: error,
                                    restricted: "Family member removal is restricted for this account.",
                                    offline: "Offline mode active. Reconnect and retry family member removal.",
                                    fallback: "Could not remove family member. Please try again.",
                                )
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("Family Members")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button { showAddSheet = true } label: { Image(systemName: "plus") }
            }
        }
        .task { await loadMembers() }
        .refreshable { await loadMembers() }
        .sheet(isPresented: $showAddSheet) {
            NavigationStack {
                AddFamilyMemberView { request in
                    Task {
                        do {
                            try await api.addFamilyMember(request: request)
                            errorMessage = nil
                            await loadMembers()
                            showAddSheet = false
                        } catch {
                            errorMessage = RetailerErrorSupport.message(
                                for: error,
                                restricted: "Family member creation is restricted for this account.",
                                offline: "Offline mode active. Reconnect and retry family member creation.",
                                fallback: "Could not add family member. Please try again.",
                            )
                        }
                    }
                }
            }
            .presentationDetents([.medium])
        }
    }
    
    private func loadMembers() async {
        isLoading = true
        errorMessage = nil
        do {
            members = try await api.getFamilyMembers()
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Family members access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry loading family members.",
                fallback: "Family members could not be loaded. Please try again.",
            )
        }
        isLoading = false
    }
}

struct AddFamilyMemberView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var nickname = ""
    var onAdd: (FamilyMemberRequest) -> Void

    var body: some View {
        Form {
            Section("Details") {
                TextField("Nickname", text: $nickname)
            }
        }
        .navigationTitle("Add Member")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
            ToolbarItem(placement: .confirmationAction) {
                Button("Save") {
                    onAdd(FamilyMemberRequest(nickname: nickname, photoUrl: nil))
                }
                .disabled(nickname.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
    }
}
