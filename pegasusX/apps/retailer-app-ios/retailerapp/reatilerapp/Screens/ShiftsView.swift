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
                Text("Clock in before POS when SHIFTS is on. Close shift with counted cash for variance alerts.")
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
                    Button("Clock out") { Task { await clockOut() } }
                        .disabled(busy)
                }
            }
            Section("Cash shift") {
                TextField("Opening float (minor)", text: $floatMinor)
                    .keyboardType(.numberPad)
                TextField("Closing cash (minor)", text: $closingCash)
                    .keyboardType(.numberPad)
                Button(busy ? "…" : "Open shift") { Task { await openShift() } }
                    .disabled(busy || !clockedIn)
            }
            Section("Shifts") {
                if shifts.isEmpty {
                    Text("No shifts yet").foregroundStyle(AppTheme.textTertiary)
                } else {
                    ForEach(shifts) { row in
                        VStack(alignment: .leading, spacing: 4) {
                            Text("\(row.status) · float \(Double(row.openingFloatMinor) / 100.0)")
                            if let v = row.varianceMinor {
                                Text(String(format: "Variance %.2f", Double(v) / 100.0))
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textSecondary)
                            }
                            if row.status == "OPEN" {
                                Button("Close shift") { Task { await closeShift(id: row.id) } }
                                    .disabled(busy)
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("Shifts")
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
