import SwiftUI

struct ShiftsView: View {
    @State private var vm = ShiftsViewModel()

    var body: some View {
        @Bindable var vm = vm
        List {
            Section {
                Text("mobile_retailer.ui.clock_in_before_pos_when_shifts_is_on_close_shift_with_counted_c")
                    .font(.system(.footnote, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
            }
            if let banner = vm.banner {
                Section {
                    Text(banner)
                        .font(.caption)
                        .foregroundStyle(AppTheme.accent)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel(banner)
                }
            }
            Section("Time clock") {
                Text(vm.clockedIn ? "Clocked in" : "Not clocked in")
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel(vm.clockedIn ? "Status: Clocked in" : "Status: Not clocked in")
                if !vm.clockedIn {
                    Button(vm.busy ? "…" : "Clock in") {
                        Task { await vm.clockIn() }
                    }
                    .pegasusMinimumHitTarget()
                    .disabled(vm.busy)
                    .accessibilityLabel("Clock in")
                } else {
                    Button("mobile_retailer.ui.clock_out") {
                        Task { await vm.clockOut() }
                    }
                    .pegasusMinimumHitTarget()
                    .disabled(vm.busy)
                    .accessibilityLabel("Clock out")
                }
            }
            Section("Cash shift") {
                TextField("mobile_retailer.ui.opening_float_minor", text: $vm.floatMinor)
                    .keyboardType(.numberPad)
                    .accessibilityLabel("Opening float minor")
                TextField("mobile_retailer.ui.closing_cash_minor", text: $vm.closingCash)
                    .keyboardType(.numberPad)
                    .accessibilityLabel("Closing cash minor")
                Button(vm.busy ? "…" : "Open shift") {
                    Task { await vm.openShift() }
                }
                .pegasusMinimumHitTarget()
                .disabled(vm.busy || !vm.clockedIn)
                .accessibilityLabel("Open shift")
            }
            Section("portal.nav.shifts") {
                if vm.shifts.isEmpty {
                    Text("mobile_retailer.ui.no_shifts_yet")
                        .foregroundStyle(AppTheme.textTertiary)
                } else {
                    ForEach(vm.shifts) { row in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(L10n.format("mobile_retailer.ui.status_float_n_0", "\(row.status)", "\(vm.formatMinor(row.openingFloatMinor))"))
                                .font(.system(.subheadline, design: .rounded))
                            if let v = row.varianceMinor {
                                Text("Variance \(vm.formatMinor(v))")
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textSecondary)
                            }
                            if row.status == "OPEN" {
                                Button("mobile_retailer.ui.close_shift") {
                                    Task { await vm.closeShift(id: row.id) }
                                }
                                .buttonStyle(.bordered)
                                .pegasusMinimumHitTarget()
                                .disabled(vm.busy)
                                .accessibilityLabel("Close shift \(row.id)")
                            }
                        }
                        .padding(.vertical, 4)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Shift \(row.status), opening float \(vm.formatMinor(row.openingFloatMinor))")
                    }
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("portal.nav.shifts")
        .navigationBarTitleDisplayMode(.inline)
        .task { await vm.refresh() }
        .refreshable { await vm.refresh() }
    }
}
