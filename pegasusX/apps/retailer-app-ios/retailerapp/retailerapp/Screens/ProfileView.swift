import SwiftUI
import UIKit

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
    @State private var pricingRulesSummary: String = ""
    @State private var orderCount: Int = 0
    @State private var totalSpent: Int64 = 0
    @State private var totalSpentCurrency: String = "UZS"
    @State private var creditProfile: CreditProfile?
    @State private var creditLoading = true
    @State private var creditMissing = false
    @State private var creditError: String?

    @Environment(AuthManager.self) private var auth

    private var user: User { auth.currentUser ?? User(id: "", name: "—", company: "—", email: "—", avatarURL: nil) }
    private var displayName: String { profileName.isEmpty ? user.name : profileName }
    private var displayCompany: String { profileCompany.isEmpty ? user.company : profileCompany }
    private let api = APIClient.shared

    var body: some View {
        ScrollView {
            VStack(spacing: AppTheme.spacingLG) {
                // User Identity Card
                UserCard(displayName: displayName, displayCompany: displayCompany, userEmail: user.email, profilePhone: profilePhone).slideIn(delay: 0)

                // Stats row
                StatsRowView(orderCount: orderCount, totalSpent: totalSpent, totalSpentCurrency: totalSpentCurrency).slideIn(delay: 0.05)

                CreditProfileSection(
                    profile: creditProfile,
                    isLoading: creditLoading,
                    missing: creditMissing,
                    error: creditError
                ).slideIn(delay: 0.06)

                // Order History link
                OrderHistoryLink(orderCount: orderCount).slideIn(delay: 0.07)

                // Empathy Engine — Global Auto-Order
                EmpathyEngineSection(globalAutoOrder: $globalAutoOrder, showHistoryAlert: $showHistoryAlert, toggleAction: { enabled, useHistory in await toggleGlobalAutoOrder(enabled: enabled, useHistory: useHistory) }).slideIn(delay: 0.09)

                if !pricingRulesSummary.isEmpty {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Pricing rules")
                            .font(.system(.subheadline, design: .rounded, weight: .bold))
                        Text(pricingRulesSummary)
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, AppTheme.spacingLG)
                }

                // Settings
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
                ]).slideIn(delay: 0.1)

                PreferencesSection(aiAutoOrder: $aiAutoOrder, notificationsEnabled: $notificationsEnabled).slideIn(delay: 0.15)

                SettingsSectionView(title: "Support", icon: "questionmark.circle.fill", items: [
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
        .task { await loadCreditProfile() }
        .task(id: refreshCenter.refreshToken) {
            await loadProfile()
            await loadStats()
            await loadCreditProfile()
        }
        .refreshable {
            await loadProfile()
            await loadStats()
            await loadCreditProfile()
        }
    }

    // MARK: - User Card

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
        do {
            struct RetailerPricingRulesResponse: Decodable {
                let summary: String?
                let status: String?
            }
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

    private func loadCreditProfile() async {
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
    @State private var familyWrites: String = "open"
    @State private var migrating = false
    @State private var showMigrateConfirm = false
    @State private var migrateResult: FamilyMigrateResult? = nil
    @State private var banner: String? = nil

    private let api = APIClient.shared
    private var familyGone: Bool { familyWrites == "gone" }

    var body: some View {
        List {
            Section {
                VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                    Label("Migrate Family → Team", systemImage: "arrow.left.arrow.right")
                        .font(.headline)
                    Text("Contacts with a phone become Team RECEIVER accounts. Temp passwords show once.")
                        .font(.caption)
                        .foregroundStyle(AppTheme.textSecondary)
                    Button {
                        showMigrateConfirm = true
                    } label: {
                        if migrating {
                            ProgressView()
                        } else {
                            Text(familyGone ? "Already migrated" : "Migrate to Team")
                        }
                    }
                    .disabled(migrating || familyGone || members.isEmpty)
                    .buttonStyle(.borderedProminent)
                }
                .padding(.vertical, 4)
            }

            if let migrateResult {
                Section("Migration result") {
                    Text("\(migrateResult.migrated.count) migrated · \(migrateResult.skipped.count) skipped · \(migrateResult.familyRemaining) remaining")
                        .font(.caption)
                        .foregroundStyle(AppTheme.textSecondary)
                    ForEach(migrateResult.migrated) { m in
                        VStack(alignment: .leading, spacing: 2) {
                            Text(m.name).font(.body.weight(.semibold))
                            Text("\(m.phone) · \(m.retailerRole)")
                                .font(.caption)
                                .foregroundStyle(AppTheme.textTertiary)
                            if let pw = m.tempPassword {
                                HStack {
                                    Text(pw).font(.system(.caption, design: .monospaced))
                                    Spacer()
                                    Button("Copy") {
                                        UIPasteboard.general.string = pw
                                    }
                                    .font(.caption)
                                }
                            }
                        }
                    }
                    ForEach(migrateResult.skipped) { s in
                        Text("\(s.phone ?? s.memberId): \(s.reason)")
                            .font(.caption)
                            .foregroundStyle(.orange)
                    }
                }
            }

            if let banner {
                Section {
                    Text(banner).font(.caption).foregroundStyle(AppTheme.accent)
                }
            }

            if let errorMessage {
                Section {
                    Text(errorMessage).font(.caption).foregroundStyle(.red)
                }
            }

            Section("Members") {
                if isLoading && members.isEmpty {
                    ProgressView()
                        .frame(maxWidth: .infinity, alignment: .center)
                } else if members.isEmpty {
                    VStack(spacing: AppTheme.spacingMD) {
                        Image(systemName: familyGone ? "person.3" : "person.2.badge.plus")
                            .font(.system(size: 36))
                            .foregroundStyle(AppTheme.textTertiary)
                        Text(familyGone ? "Family list empty" : "No Family Members")
                            .font(.headline)
                        Text(familyGone
                            ? "Use Team to manage staff."
                            : "Add members with a phone, then migrate to Team.")
                            .font(.subheadline)
                            .foregroundStyle(AppTheme.textSecondary)
                            .multilineTextAlignment(.center)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 24)
                } else {
                    ForEach(members) { member in
                        HStack(spacing: AppTheme.spacingMD) {
                            ZStack {
                                Circle().fill(AppTheme.surfaceElevated).frame(width: 40, height: 40)
                                Image(systemName: "person.fill").foregroundStyle(AppTheme.accent)
                            }
                            VStack(alignment: .leading, spacing: 2) {
                                Text(member.name).font(.system(.body, design: .rounded, weight: .semibold))
                                if let phone = member.phone, !phone.isEmpty {
                                    Text(phone).font(.caption).foregroundStyle(AppTheme.textTertiary)
                                } else {
                                    Text("No phone — skipped on migrate")
                                        .font(.caption)
                                        .foregroundStyle(.orange)
                                }
                                if let created = member.createdAt {
                                    Text("Added \(created.prefix(10))")
                                        .font(.caption2)
                                        .foregroundStyle(AppTheme.textTertiary)
                                }
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
        }
        .navigationTitle("Family Members")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                if !familyGone {
                    Button { showAddSheet = true } label: { Image(systemName: "plus") }
                }
            }
        }
        .task { await loadMembers() }
        .refreshable { await loadMembers() }
        .confirmationDialog("Migrate to Team?", isPresented: $showMigrateConfirm, titleVisibility: .visible) {
            Button("Migrate") {
                Task { await migrateToTeam() }
            }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("Temporary passwords appear once. Family add closes after migrate.")
        }
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
                            if case APIError.serverError(let code, _) = error, code == 410 {
                                familyWrites = "gone"
                                errorMessage = "Family writes closed. Use Team staff."
                            } else {
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
            }
            .presentationDetents([.medium])
        }
    }

    private func loadMembers() async {
        isLoading = true
        errorMessage = nil
        do {
            let list = try await api.getFamilyMembersList()
            members = list.members
            familyWrites = list.familyWrites ?? "open"
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

    private func migrateToTeam() async {
        migrating = true
        errorMessage = nil
        do {
            let result = try await api.migrateFamilyToTeam()
            migrateResult = result
            familyWrites = result.familyWrites
            banner = "Migrated \(result.migrated.count). Copy temp passwords now."
            await loadMembers()
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Migration requires staff.manage permission.",
                offline: "Offline mode active. Reconnect and retry migration.",
                fallback: "Migration failed. Please try again.",
            )
        }
        migrating = false
    }
}

struct AddFamilyMemberView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var name = ""
    @State private var phone = ""
    var onAdd: (FamilyMemberRequest) -> Void

    var body: some View {
        Form {
            Section("Details") {
                TextField("Name", text: $name)
                TextField("Phone (required for Team migrate)", text: $phone)
                    .keyboardType(.phonePad)
            }
        }
        .navigationTitle("Add Member")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
            ToolbarItem(placement: .confirmationAction) {
                Button("Save") {
                    onAdd(FamilyMemberRequest(
                        name: name.trimmingCharacters(in: .whitespacesAndNewlines),
                        phone: phone.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                            ? nil
                            : phone.trimmingCharacters(in: .whitespacesAndNewlines),
                        photoUrl: nil
                    ))
                }
                .disabled(name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
    }
}
