import SwiftUI

struct ShiftsView: View {
    @State private var clockedIn = false
    @State private var shifts: [ShiftRow] = []
    @State private var floatMinor = "0"
    @State private var closingCash = "0"
    @State private var registerId: String?
    @State private var banner: String?
    @State private var busy = false

    private let api = APIClient.shared

    var body: some View {
        List {
            Section {
                Text("mobile_retailer.ui.clock_in_before_pos_when_shifts_is_on_close_shift_with_counted_c")
                    .font(.system(.footnote, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
            }
            if let banner {
                Section { Text(banner).font(.caption).foregroundStyle(AppTheme.accent) }
            }
            Section("Time clock") {
                Text(clockedIn ? "Clocked in" : "Not clocked in")
                if !clockedIn {
                    Button(busy ? "…" : "Clock in") { Task { await clockIn() } }
                        .disabled(busy)
                } else {
                    Button("mobile_retailer.ui.clock_out") { Task { await clockOut() } }
                        .disabled(busy)
                }
            }
            Section("Cash shift") {
                TextField("mobile_retailer.ui.opening_float_minor", text: $floatMinor)
                    .keyboardType(.numberPad)
                TextField("mobile_retailer.ui.closing_cash_minor", text: $closingCash)
                    .keyboardType(.numberPad)
                Button(busy ? "…" : "Open shift") { Task { await openShift() } }
                    .disabled(busy || !clockedIn)
            }
            Section("portal.nav.shifts") {
                if shifts.isEmpty {
                    Text("mobile_retailer.ui.no_shifts_yet").foregroundStyle(AppTheme.textTertiary)
                } else {
                    ForEach(shifts) { row in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(L10n.format("mobile_retailer.ui.status_float_n_0", "\(row.status)", "\(Double(row.openingFloatMinor) / 100.0)"))
                            if let v = row.varianceMinor {
                                Text(String(format: "Variance %.2f", Double(v) / 100.0))
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textSecondary)
                            }
                            if row.status == "OPEN" {
                                Button("mobile_retailer.ui.close_shift") { Task { await closeShift(id: row.id) } }
                                    .disabled(busy)
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("portal.nav.shifts")
        .navigationBarTitleDisplayMode(.inline)
        .task { await refresh() }
    }

    private func refresh() async {
        do {
            let time = try await api.getTimeEntries()
            clockedIn = time.clockedIn
            let list = try await api.getShifts()
            shifts = list.items.map {
                ShiftRow(
                    id: $0.shiftId,
                    status: $0.status,
                    openingFloatMinor: $0.openingFloatMinor,
                    varianceMinor: $0.varianceMinor
                )
            }
            if registerId == nil {
                let regs = try await api.getRegisters()
                registerId = regs.items.first?.registerId
            }
        } catch {
            banner = error.localizedDescription
        }
    }

    private func clockIn() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.clockIn()
            banner = "Clocked in"
            await refresh()
        } catch {
            banner = error.localizedDescription
        }
    }

    private func clockOut() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.clockOut()
            banner = "Clocked out"
            await refresh()
        } catch {
            banner = error.localizedDescription
        }
    }

    private func openShift() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.openShift(
                registerId: registerId,
                openingFloatMinor: Int64(floatMinor) ?? 0
            )
            banner = "Shift opened"
            await refresh()
        } catch {
            banner = error.localizedDescription
        }
    }

    private func closeShift(id: String) async {
        busy = true
        defer { busy = false }
        do {
            let closed = try await api.closeShift(
                shiftId: id,
                closingCashMinor: Int64(closingCash) ?? 0
            )
            if let v = closed.varianceMinor {
                banner = String(format: "Closed · variance %.2f", Double(v) / 100.0)
            } else {
                banner = "Shift closed"
            }
            await refresh()
        } catch {
            banner = error.localizedDescription
        }
    }
}

struct ShiftRow: Identifiable {
    let id: String
    let status: String
    let openingFloatMinor: Int64
    let varianceMinor: Int64?
}
