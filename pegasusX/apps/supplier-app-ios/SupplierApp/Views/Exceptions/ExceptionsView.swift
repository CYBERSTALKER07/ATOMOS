import SwiftUI

struct ExceptionsView: View {
    @State private var rows: [SupplierExceptionRow] = []
    @State private var loading = true
    @State private var error: String?

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
                ExceptionsList(rows: rows)
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
}
