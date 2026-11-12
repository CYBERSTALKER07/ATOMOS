import Foundation
import Observation

@Observable
@MainActor
final class CapabilitiesViewModel {
    var packs: [RetailerCapabilityPack] = []
    var loading = true
    var errorText: String?
    var banner: String?
    var busyId: String?

    private let api = APIClient.shared

    func load() async {
        loading = true
        errorText = nil
        defer { loading = false }
        do {
            let res = try await api.getCapabilities()
            packs = res.packs
        } catch {
            errorText = error.localizedDescription
        }
    }

    func enable(_ id: String) async {
        busyId = id
        defer { busyId = nil }
        do {
            let res = try await api.enableCapability(packId: id, acceptSoft: true, enableDeps: true)
            banner = res.message ?? "\(id) enabled"
            Haptics.success()
            await load()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }

    func disable(_ id: String) async {
        busyId = id
        defer { busyId = nil }
        do {
            _ = try await api.disableCapability(packId: id)
            banner = "\(id) disabled"
            Haptics.selection()
            await load()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }
}
