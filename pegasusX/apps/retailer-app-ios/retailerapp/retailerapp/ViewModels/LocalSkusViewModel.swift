import Foundation
import Observation

struct LocalSkuRowIOS: Identifiable {
    let id: String
    let name: String
    let priceMinor: Int64
    let currency: String
    let active: Bool
}

@Observable
@MainActor
final class LocalSkusViewModel {
    var rows: [LocalSkuRowIOS] = []
    var name = ""
    var barcode = ""
    var priceInput = "5000"
    var banner: String?
    var loadError: String?
    var summaryReady = false
    var busy = false

    private let api = APIClient.shared

    var canCreate: Bool {
        !busy && !name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    func load() async {
        do {
            let list = try await api.getLocalSkus()
            rows = list.items.map {
                LocalSkuRowIOS(
                    id: $0.localSkuId,
                    name: $0.name,
                    priceMinor: $0.defaultPriceMinor ?? 0,
                    currency: displayPackCurrency($0.currency),
                    active: $0.isActive ?? true
                )
            }
            loadError = nil
            summaryReady = true
        } catch {
            rows = []
            summaryReady = false
            loadError = error.localizedDescription
        }
    }

    func create() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.createLocalSku(
                name: name.trimmingCharacters(in: .whitespacesAndNewlines),
                barcode: barcode.trimmingCharacters(in: .whitespacesAndNewlines),
                priceMinor: Int64(priceInput.trimmingCharacters(in: .whitespacesAndNewlines)) ?? 0
            )
            name = ""
            barcode = ""
            banner = "Created"
            Haptics.success()
            await load()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }

    func toggle(_ row: LocalSkuRowIOS) async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.patchLocalSku(id: row.id, isActive: !row.active)
            Haptics.selection()
            await load()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }
}
