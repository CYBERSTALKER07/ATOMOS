import re

with open("pegasus/apps/retailer-app-ios/retailerapp/reatilerapp/Screens/ProfileView.swift", "r") as f:
    content = f.read()

# 1. Add "SettingsItem(icon: "person.2.fill", title: "Family Members", subtitle: "Manage your family / staff")," to Company array
part1 = re.sub(
    r'(SettingsItem\(icon: "key", title: "API Access", subtitle: "Developer settings"\),)',
    r'\1\n                    SettingsItem(icon: "person.2.fill", title: "Family Members", subtitle: "Manage family/staff", view: "FamilyMembers"),',
    content
)

# 2. Add 'let view: String?' to SettingsItem
part2 = re.sub(
    r'let subtitle: String\?',
    r'let subtitle: String?\n    var view: String? = nil',
    part1
)

# 3. Change settingsRow to use NavigationLink if view is provided
row_replacement = """    private func settingsRow(_ item: SettingsItem) -> some View {
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
    
    private func settingsRowContent(_ item: SettingsItem) -> some View {"""

part3 = part2.replace("    private func settingsRow(_ item: SettingsItem) -> some View {", row_replacement)


# Append FamilyMembersView at the end.
appendage = """

// MARK: - Family Members
struct FamilyMembersView: View {
    @State private var members: [FamilyMemberResponse] = []
    @State private var isLoading = false
    @State private var showAddSheet = false

    private let api = APIClient.shared

    var body: some View {
        List {
            if isLoading && members.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, alignment: .center)
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
                            Text("Added \\(member.createdAt.prefix(10))").font(.caption).foregroundStyle(AppTheme.textTertiary)
                        }
                    }
                    .padding(.vertical, 4)
                }
                .onDelete { offsets in
                    let membersToDelete = offsets.map { members[$0] }
                    for m in membersToDelete {
                        Task {
                            do { try await api.removeFamilyMember(memberId: m.id); await loadMembers() }
                            catch { print("Delete failed") }
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
                        do { try await api.addFamilyMember(request: request); await loadMembers(); showAddSheet = false }
                        catch { print("Add failed") }
                    }
                }
            }
            .presentationDetents([.medium])
        }
    }
    
    private func loadMembers() async {
        isLoading = true
        do { members = try await api.getFamilyMembers() } catch { print("Load failed: \\(error)") }
        isLoading = false
    }
}

struct AddFamilyMemberView: View {
    @Environment(\\.dismiss) private var dismiss
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
"""

with open("pegasus/apps/retailer-app-ios/retailerapp/reatilerapp/Screens/ProfileView.swift", "w") as f:
    f.write(part3 + appendage)

