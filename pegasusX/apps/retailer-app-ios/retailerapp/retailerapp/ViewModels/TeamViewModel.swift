import Foundation
import Observation

@Observable
@MainActor
final class TeamViewModel {
    var members: [RetailerOrgMember] = []
    var loading = true
    var errorText: String?
    var banner: String?
    var name = ""
    var phone = ""
    var password = ""
    var role = "CASHIER"
    var busy = false

    private let api = APIClient.shared

    func load() async {
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

    func invite() async {
        busy = true
        defer { busy = false }
        do {
            let res = try await api.createOrgMember(
                name: name.trimmingCharacters(in: .whitespacesAndNewlines),
                phone: phone.trimmingCharacters(in: .whitespacesAndNewlines),
                password: password,
                role: role.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
            )
            members = res.items
            banner = "Member created"
            name = ""
            phone = ""
            password = ""
            Haptics.success()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }

    func deactivate(_ userId: String) async {
        busy = true
        defer { busy = false }
        do {
            let res = try await api.deactivateOrgMember(userId: userId)
            members = res.items
            banner = "Deactivated"
            Haptics.selection()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }
}
