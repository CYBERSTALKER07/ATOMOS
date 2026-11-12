import Foundation
import Observation

@Observable
@MainActor
final class LocationsViewModel {
    var items: [RetailerLocationDTO] = []
    var activeId = ""
    var loading = true
    var banner: String?
    var name = ""
    var address = ""
    var busy = false

    private let api = APIClient.shared

    var canCreate: Bool {
        !busy && !name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    func load() async {
        loading = true
        defer { loading = false }
        do {
            let res = try await api.getLocations()
            items = res.items
            activeId = res.activeLocationId ?? ""
        } catch {
            banner = error.localizedDescription
        }
    }

    func create() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.createLocation(
                name: name.trimmingCharacters(in: .whitespacesAndNewlines),
                address: address.trimmingCharacters(in: .whitespacesAndNewlines)
            )
            name = ""
            address = ""
            banner = "Created"
            Haptics.success()
            await load()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }

    func switchTo(_ id: String) async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.switchLocation(locationId: id)
            activeId = id
            banner = "Switched active branch"
            Haptics.selection()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }

    func setPrimary(_ id: String) async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.setPrimaryLocation(locationId: id)
            banner = "Primary updated"
            Haptics.success()
            await load()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }
}
