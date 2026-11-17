import Foundation
import Observation
import UIKit

@Observable
@MainActor
final class FamilyMembersViewModel {
    var members: [FamilyMemberResponse] = []
    var isLoading = false
    var showAddSheet = false
    var errorMessage: String?
    var familyWrites = "open"
    var migrating = false
    var showMigrateConfirm = false
    var migrateResult: FamilyMigrateResult?
    var banner: String?

    private let api = APIClient.shared

    var familyGone: Bool {
        familyWrites == "gone"
    }

    func copyTempPassword(_ password: String) {
        UIPasteboard.general.string = password
        Haptics.selection()
    }

    func loadMembers() async {
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
                fallback: "Family members could not be loaded. Please try again."
            )
        }
        isLoading = false
    }

    func migrateToTeam() async {
        migrating = true
        errorMessage = nil
        do {
            let result = try await api.migrateFamilyToTeam()
            migrateResult = result
            familyWrites = result.familyWrites
            banner = "Migrated \(result.migrated.count). Copy temp passwords now."
            Haptics.success()
            await loadMembers()
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Migration requires staff.manage permission.",
                offline: "Offline mode active. Reconnect and retry migration.",
                fallback: "Migration failed. Please try again."
            )
            Haptics.error()
        }
        migrating = false
    }

    func addMember(request: FamilyMemberRequest) async {
        do {
            try await api.addFamilyMember(request: request)
            errorMessage = nil
            Haptics.success()
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
                    fallback: "Could not add family member. Please try again."
                )
            }
            Haptics.error()
        }
    }

    func removeMembers(at offsets: IndexSet) async {
        let membersToDelete = offsets.map { members[$0] }
        for member in membersToDelete {
            do {
                try await api.removeFamilyMember(memberId: member.id)
                errorMessage = nil
                Haptics.selection()
            } catch {
                errorMessage = RetailerErrorSupport.message(
                    for: error,
                    restricted: "Family member removal is restricted for this account.",
                    offline: "Offline mode active. Reconnect and retry family member removal.",
                    fallback: "Could not remove family member. Please try again."
                )
                Haptics.error()
            }
        }
        await loadMembers()
    }
}
