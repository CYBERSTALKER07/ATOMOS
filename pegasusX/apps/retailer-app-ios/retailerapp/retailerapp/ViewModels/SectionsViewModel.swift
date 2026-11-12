import Foundation
import Observation

struct SectionRowIOS: Identifiable {
    let id: String
    let name: String
}

@Observable
@MainActor
final class SectionsViewModel {
    var items: [SectionRowIOS] = []
    var name = "Dairy"
    var aisle = ""
    var selectedId: String?
    var skuText = ""
    var banner: String?
    var saveError: String?
    var busy = false

    private let api = APIClient.shared

    func refresh() async {
        do {
            let res = try await api.getSections()
            items = res.items.map { SectionRowIOS(id: $0.sectionId, name: $0.name) }
        } catch {
            banner = error.localizedDescription
        }
    }

    func create() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.createSection(name: name, aisleTag: aisle.isEmpty ? nil : aisle)
            saveError = nil
            banner = "Created"
            Haptics.success()
            await refresh()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }

    func saveSkus() async {
        guard let selectedId else { return }
        busy = true
        defer { busy = false }
        do {
            let skus = skuText.split { $0 == "," || $0.isWhitespace }.map(String.init).filter { !$0.isEmpty }
            _ = try await api.putSectionSkus(sectionId: selectedId, skus: skus)
            saveError = nil
            banner = "SKUs saved"
            Haptics.success()
        } catch {
            saveError = "section_skus_failed"
            Haptics.error()
        }
    }
}
