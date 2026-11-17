import SwiftUI

struct FamilyMembersView: View {
    @State private var vm = FamilyMembersViewModel()

    var body: some View {
        @Bindable var vm = vm
        List {
            Section {
                VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                    Label("Migrate Family → Team", systemImage: "arrow.left.arrow.right")
                        .font(.headline)
                    Text("Contacts with a phone become Team RECEIVER accounts. Temp passwords show once.")
                        .font(.caption)
                        .foregroundStyle(AppTheme.textSecondary)
                    Button {
                        Haptics.light()
                        vm.showMigrateConfirm = true
                    } label: {
                        if vm.migrating {
                            ProgressView()
                        } else {
                            Text(vm.familyGone ? "Already migrated" : "Migrate to Team")
                        }
                    }
                    .pegasusMinimumHitTarget()
                    .disabled(vm.migrating || vm.familyGone || vm.members.isEmpty)
                    .buttonStyle(.borderedProminent)
                    .accessibilityLabel(vm.familyGone ? "Already migrated to team" : "Migrate family contacts to team")
                }
                .padding(.vertical, 4)
            }

            if let migrateResult = vm.migrateResult {
                Section("Migration result") {
                    Text("\(migrateResult.migrated.count) migrated · \(migrateResult.skipped.count) skipped · \(migrateResult.familyRemaining) remaining")
                        .font(.caption)
                        .foregroundStyle(AppTheme.textSecondary)
                    ForEach(migrateResult.migrated) { member in
                        VStack(alignment: .leading, spacing: 2) {
                            Text(member.name)
                                .font(.body.weight(.semibold))
                            Text("\(member.phone) · \(member.retailerRole)")
                                .font(.caption)
                                .foregroundStyle(AppTheme.textTertiary)
                            if let pw = member.tempPassword {
                                HStack {
                                    Text(pw)
                                        .font(.system(.caption, design: .monospaced))
                                    Spacer()
                                    Button("Copy") {
                                        vm.copyTempPassword(pw)
                                    }
                                    .buttonStyle(.bordered)
                                    .font(.caption)
                                    .pegasusMinimumHitTarget()
                                    .accessibilityLabel("Copy temporary password for \(member.name)")
                                }
                            }
                        }
                    }
                    ForEach(migrateResult.skipped) { skipped in
                        Text("\(skipped.phone ?? skipped.memberId): \(skipped.reason)")
                            .font(.caption)
                            .foregroundStyle(.orange)
                    }
                }
            }

            if let banner = vm.banner {
                Section {
                    Text(banner)
                        .font(.caption)
                        .foregroundStyle(AppTheme.accent)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel(banner)
                }
            }

            if let errorMessage = vm.errorMessage {
                Section {
                    Text(errorMessage)
                        .font(.caption)
                        .foregroundStyle(AppTheme.destructive)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Error: \(errorMessage)")
                }
            }

            Section("Members") {
                if vm.isLoading && vm.members.isEmpty {
                    ProgressView()
                        .frame(maxWidth: .infinity, alignment: .center)
                } else if vm.members.isEmpty {
                    VStack(spacing: AppTheme.spacingMD) {
                        Image(systemName: vm.familyGone ? "person.3" : "person.2.badge.plus")
                            .font(.largeTitle)
                            .foregroundStyle(AppTheme.textTertiary)
                        Text(vm.familyGone ? "Family list empty" : "No Family Members")
                            .font(.headline)
                        Text(vm.familyGone
                            ? "Use Team to manage staff."
                            : "Add members with a phone, then migrate to Team.")
                            .font(.subheadline)
                            .foregroundStyle(AppTheme.textSecondary)
                            .multilineTextAlignment(.center)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 24)
                    .accessibilityElement(children: .combine)
                } else {
                    ForEach(vm.members) { member in
                        HStack(spacing: AppTheme.spacingMD) {
                            ZStack {
                                Circle().fill(AppTheme.surfaceElevated).frame(width: 40, height: 40)
                                Image(systemName: "person.fill").foregroundStyle(AppTheme.accent)
                            }
                            VStack(alignment: .leading, spacing: 2) {
                                Text(member.name)
                                    .font(.system(.body, design: .rounded, weight: .semibold))
                                if let phone = member.phone, !phone.isEmpty {
                                    Text(phone)
                                        .font(.caption)
                                        .foregroundStyle(AppTheme.textTertiary)
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
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("\(member.name), \(member.phone ?? "no phone")")
                    }
                    .onDelete { offsets in
                        Task { await vm.removeMembers(at: offsets) }
                    }
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("Family Members")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                if !vm.familyGone {
                    Button {
                        Haptics.light()
                        vm.showAddSheet = true
                    } label: {
                        Image(systemName: "plus")
                            .pegasusMinimumHitTarget()
                    }
                    .accessibilityLabel("Add family member")
                }
            }
        }
        .task { await vm.loadMembers() }
        .refreshable { await vm.loadMembers() }
        .confirmationDialog("Migrate to Team?", isPresented: $vm.showMigrateConfirm, titleVisibility: .visible) {
            Button("Migrate") {
                Task { await vm.migrateToTeam() }
            }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("Temporary passwords appear once. Family add closes after migrate.")
        }
        .sheet(isPresented: $vm.showAddSheet) {
            NavigationStack {
                AddFamilyMemberView { request in
                    Task { await vm.addMember(request: request) }
                }
            }
            .presentationDetents([.medium])
        }
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
                    .accessibilityLabel("Member name")
                TextField("Phone (required for Team migrate)", text: $phone)
                    .keyboardType(.phonePad)
                    .accessibilityLabel("Member phone number")
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("Add Member")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button("Cancel") { dismiss() }
                    .pegasusMinimumHitTarget()
            }
            ToolbarItem(placement: .confirmationAction) {
                Button("Save") {
                    let trimmedPhone = phone.trimmingCharacters(in: .whitespacesAndNewlines)
                    onAdd(FamilyMemberRequest(
                        name: name.trimmingCharacters(in: .whitespacesAndNewlines),
                        phone: trimmedPhone.isEmpty ? nil : trimmedPhone,
                        photoUrl: nil
                    ))
                }
                .pegasusMinimumHitTarget()
                .disabled(name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
    }
}
