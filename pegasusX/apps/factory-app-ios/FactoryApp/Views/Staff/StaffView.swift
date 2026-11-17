import SwiftUI

struct StaffView: View {
    @State private var vm = StaffViewModel()

    var body: some View {
        Group {
            if vm.loading {
                FactoryLoadingView(
                    title: "Loading staff",
                    message: "Fetching factory operators and shift status."
                )
            } else if let error = vm.error, vm.staff.isEmpty {
                FactoryErrorView(message: error, retry: { Task { await vm.load() } })
            } else if vm.staff.isEmpty {
                FactoryStateView(
                    kind: .empty,
                    headline: "No staff",
                    message: "No staff members are registered for this factory.",
                    actionTitle: "Add Staff",
                    action: {
                        PegasusHaptics.selection()
                        vm.presentCreate()
                    }
                )
            } else {
                StaffListContent(staff: vm.staff)
                    .navigationDestination(for: String.self) { staffId in
                        StaffDetailView(staffId: staffId)
                    }
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.staff")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                } label: {
                    Image(systemName: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh staff")
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.selection()
                    vm.presentCreate()
                } label: {
                    Image(systemName: "plus")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Add staff member")
            }
        }
        .task {
            vm.startRealtime()
            await vm.load()
        }
        .onDisappear {
            vm.stopRealtime()
        }
        .sheet(isPresented: $vm.showCreate) {
            CreateStaffSheet(vm: vm)
        }
    }
}
