import SwiftUI

/// Retail OS Phase 1 — team roster + invite.
struct TeamView: View {
    @State private var members: [RetailerOrgMember] = []
    @State private var loading = true
    @State private var errorText: String?
    @State private var banner: String?
    @State private var name = ""
    @State private var phone = ""
    @State private var password = ""
    @State private var role = "CASHIER"
    @State private var busy = false

    private let api = APIClient.shared

    var body: some View {
        List {
            Section {
                Text("Invite staff with roles. First invite enables the TEAM pack.")
                    .font(.system(.footnote, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
            }
            if let banner {
                Section { Text(banner).font(.caption).foregroundStyle(AppTheme.accent) }
            }
            if let errorText {
                Section {
                    Text(errorText).foregroundStyle(.red)
                    Button("Retry") { Task { await load() } }
                }
            }
            Section("Invite") {
                TextField("Name", text: $name)
                TextField("Phone", text: $phone)
                    .textContentType(.telephoneNumber)
                SecureField("Password", text: $password)
                TextField("Role (e.g. CASHIER)", text: $role)
                Button(busy ? "…" : "Create member") {
                    Task { await invite() }
                }
                .disabled(busy)
            }
            Section("Roster") {
                if loading && members.isEmpty {
                    ProgressView()
                }
                ForEach(members) { m in
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text(m.name).font(.headline)
                            if m.isOwner {
                                Text("Owner").font(.caption2).foregroundStyle(AppTheme.textTertiary)
                            }
                            if !m.isActive {
                                Text("Inactive").font(.caption2).foregroundStyle(.red)
                            }
                        }
                        Text("\(m.phone) · \(m.retailerRole)")
                            .font(.caption)
                            .foregroundStyle(AppTheme.textSecondary)
                        if !m.isOwner && m.isActive {
                            Button("Deactivate", role: .destructive) {
                                Task { await deactivate(m.userId) }
                            }
                            .font(.caption)
                        }
                    }
                    .padding(.vertical, 4)
                }
            }
        }
        .navigationTitle("Team")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private func load() async {
        loading = true
        errorText = nil
        defer { loading = false }
        do {
            let res = try await api.getOrgMembers()
            members = res.items
        } catch {
            errorText = error.localizedDescription
        }
    }

    private func invite() async {
        busy = true
        defer { busy = false }
        do {
            let res = try await api.createOrgMember(
                name: name,
                phone: phone,
                password: password,
                role: role.uppercased()
            )
            members = res.items
            banner = "Member created"
            name = ""; phone = ""; password = ""
        } catch {
            banner = error.localizedDescription
        }
    }

    private func deactivate(_ userId: String) async {
        do {
            let res = try await api.deactivateOrgMember(userId: userId)
            members = res.items
            banner = "Deactivated"
        } catch {
            banner = error.localizedDescription
        }
    }
}
