import SwiftUI

struct SupplyRequestsHubView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var vm = SupplyRequestsHubViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.requests.isEmpty {
                FactoryLoadingView(
                    title: "Loading supply requests",
                    message: "Fetching inbound supply pipeline status."
                )
            } else if let error = vm.error {
                FactoryErrorView(message: error) {
                    Task { await vm.load() }
                }
            } else if vm.requests.isEmpty {
                FactoryStateView(
                    kind: .empty,
                    headline: "No requests",
                    message: "No supply requests pending from warehouses."
                )
            } else {
                List {
                    Section {
                        FactorySectionHeader(
                            title: "Inbound supply requests",
                            subtitle: "\(vm.requests.count) active requests"
                        )
                        .listRowInsets(EdgeInsets(top: 8, leading: 0, bottom: 8, trailing: 0))
                        .listRowBackground(Color.clear)
                    }

                    Section {
                        ForEach(vm.requests) { request in
                            NavigationLink {
                                SupplyRequestDetailView(requestId: request.id)
                            } label: {
                                SupplyRequestRow(request: request)
                            }
                        }
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("portal.nav.supply_requests")
        .toolbar {
            ToolbarItem(placement: .topBarLeading) {
                Button {
                    PegasusHaptics.selection()
                    dismiss()
                } label: {
                    Image(systemName: "xmark")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Close supply requests hub")
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                } label: {
                    Image(systemName: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh supply requests")
            }
        }
        .task { await vm.load() }
    }
}
