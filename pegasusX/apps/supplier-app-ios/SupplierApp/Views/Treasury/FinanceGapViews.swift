import SwiftUI

struct CashReconciliationsView: View {
    @State private var rows: [CashReconciliationRow] = []
    @State private var loading = true

    var body: some View {
        List(rows) { row in
            VStack(alignment: .leading) {
                Text(row.reconciliationId).font(.caption.monospaced())
                Text("Driver \(row.driverId) · diff \(row.differenceMinor)")
            }
        }
        .navigationTitle("Cash reconciliations")
        .task {
            loading = true
            defer { loading = false }
            if let resp = try? await SupplierOperationsService.cashReconciliations() {
                rows = resp.reconciliations
            }
        }
    }
}

struct CreditNotesListView: View {
    @State private var rows: [CreditNoteRow] = []
    var body: some View {
        List(rows) { row in
            Text("\(row.creditNoteId) · \(row.orderId)")
        }
        .navigationTitle("Credit notes")
        .task {
            if let resp = try? await SupplierOperationsService.creditNotes() {
                rows = resp.creditNotes
            }
        }
    }
}

struct RoutePerformanceListView: View {
    @State private var rows: [RoutePerformanceRow] = []
    var body: some View {
        List(rows) { row in
            Text("Route \(row.routeId) · \(row.ordersCompleted) orders")
        }
        .navigationTitle("Route performance")
        .task {
            if let resp = try? await SupplierOperationsService.routePerformance() {
                rows = resp.routes
            }
        }
    }
}
