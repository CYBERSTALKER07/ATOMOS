import Foundation
import Observation

@Observable
@MainActor
final class CreditPartnersViewModel {
    var relationships: [CreditRelationshipWire] = []
    var invoices: [ArInvoiceWire] = []
    var banner: String?
    var isLoading = false

    private let api = APIClient.shared

    func load() async {
        isLoading = true
        defer { isLoading = false }
        do {
            async let rels = api.getCreditRelationships()
            async let invs = api.getArInvoices(status: "OPEN")
            relationships = try await rels
            invoices = try await invs
            banner = nil
        } catch {
            banner = error.localizedDescription
        }
    }
}
