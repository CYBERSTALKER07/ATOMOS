import SwiftUI

struct ExceptionsView: View {
    @State private var rows: [SupplierExceptionRow] = []
    @State private var loading = true
    @State private var error: String?
    @State private var busyKey: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading exceptions…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if rows.isEmpty {
                SupplierEmptyView(
                    title: "No exceptions",
                    message: "Operational exceptions will appear here. Use Claims for post-delivery OS&D."
                )
            } else {
                ExceptionsList(rows: rows, busyKey: busyKey) { row in
                    Task { await resolve(row) }
                }
            }
        }
        .navigationTitle("Exceptions")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                NavigationLink {
                    ClaimsView()
                } label: {
                    Text("Claims")
                }
            }
        }
        .refreshable { await load() }
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            rows = try await SupplierOperationsService.exceptions()
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func resolve(_ row: SupplierExceptionRow) async {
        let kind = row.kind.uppercased()
        let key = "\(kind):\(row.orderId)"
        busyKey = key
        defer { busyKey = nil }
        do {
            let resolveId: String
            let creditNoteId: String?
            if kind == "CREDIT_NOTE_DRAFT" {
                creditNoteId = row.note.flatMap { $0.isEmpty ? nil : $0 }
                resolveId = creditNoteId ?? row.orderId
            } else {
                creditNoteId = nil
                resolveId = row.orderId
            }
            try await SupplierOperationsService.resolveException(
                kind: kind,
                id: resolveId,
                creditNoteId: creditNoteId
            )
            await load()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
