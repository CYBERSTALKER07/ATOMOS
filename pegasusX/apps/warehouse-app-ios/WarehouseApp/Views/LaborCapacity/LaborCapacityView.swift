import SwiftUI

struct LaborCapacityView: View {
    @State private var date = Self.todayUTC()
    @State private var zones: [LaborZoneCapacity] = []
    @State private var loading = true
    @State private var error: String?
    @State private var driverId = ""
    @State private var score: LaborDriverScore?
    @State private var availHours = "8"
    @State private var availStatus = "AVAILABLE"
    @State private var zoneH3 = ""
    @State private var saving = false
    @State private var statusMessage: String?

    private static func todayUTC() -> String {
        let f = DateFormatter()
        f.calendar = Calendar(identifier: .gregorian)
        f.locale = Locale(identifier: "en_US_POSIX")
        f.timeZone = TimeZone(secondsFromGMT: 0)
        f.dateFormat = "yyyy-MM-dd"
        return f.string(from: Date())
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text("Zone delivery capacity and driver reliability scores.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                    TextField("Date (YYYY-MM-DD)", text: $date)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    Button("Refresh zones") { Task { await loadZones() } }
                        .disabled(loading)
                }

                if let error {
                    Section { Text(error).foregroundStyle(.red) }
                }
                if let statusMessage {
                    Section { Text(statusMessage).foregroundStyle(.secondary) }
                }

                Section("Zones") {
                    if loading {
                        ProgressView()
                    } else if zones.isEmpty {
                        Text("No zone capacity rows. Workers populate ZoneCapacity after availability is set.")
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(zones) { z in
                            let util = z.totalCapacity > 0 ? (z.usedCapacity / z.totalCapacity) * 100 : 0
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(z.zoneH3).font(.headline).fontDesign(.monospaced)
                                Text(String(format: "Total %.1f · Used %.1f · %.0f%%", z.totalCapacity, z.usedCapacity, util))
                                if !z.date.isEmpty {
                                    Text(z.date).font(.caption).foregroundStyle(.secondary)
                                }
                            }
                            .padding(.vertical, 2)
                        }
                    }
                }

                Section("Driver score & availability") {
                    TextField("Driver ID", text: $driverId)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    Button("Load score") { Task { await loadScore() } }
                    if let score {
                        Text(String(format: "Score %.1f", score.score)).font(.headline)
                        Text(String(
                            format: "On-time %.0f%% · Completion %.0f%% · Stops/hr %.1f",
                            score.onTimeRate * 100,
                            score.completionRate * 100,
                            score.stopsPerHour
                        ))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    }
                    TextField("Hours", text: $availHours)
                        .keyboardType(.decimalPad)
                    TextField("Status (AVAILABLE / LIMITED / OFF)", text: $availStatus)
                        .textInputAutocapitalization(.characters)
                        .autocorrectionDisabled()
                    TextField("Zone H3 (optional)", text: $zoneH3)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    Button(saving ? "Saving…" : "Save availability") {
                        Task { await saveAvailability() }
                    }
                    .disabled(saving)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Labor capacity")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                        Task { await loadZones() }
                    }
                }
            }
            .task { await loadZones() }
            .refreshable { await loadZones() }
        }
    }

    private func loadZones() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await WarehouseService.laborZoneCapacity(date: date.trimmingCharacters(in: .whitespacesAndNewlines))
            zones = resp.zones
        } catch {
            zones = []
            self.error = error.localizedDescription
        }
    }

    private func loadScore() async {
        let id = driverId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !id.isEmpty else {
            error = "Driver ID required"
            return
        }
        do {
            score = try await WarehouseService.laborDriverScore(driverId: id)
            error = nil
        } catch {
            score = nil
            self.error = "Driver score not found"
        }
    }

    private func saveAvailability() async {
        let id = driverId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !id.isEmpty else {
            error = "Driver ID required"
            return
        }
        saving = true
        statusMessage = nil
        defer { saving = false }
        do {
            let body = LaborDriverAvailabilityRequest(
                driverId: id,
                date: date.trimmingCharacters(in: .whitespacesAndNewlines),
                availableHours: Double(availHours.trimmingCharacters(in: .whitespacesAndNewlines)) ?? 0,
                status: availStatus.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                    ? "AVAILABLE"
                    : availStatus.trimmingCharacters(in: .whitespacesAndNewlines),
                zoneH3: zoneH3.trimmingCharacters(in: .whitespacesAndNewlines)
            )
            _ = try await WarehouseService.setLaborDriverAvailability(body)
            statusMessage = "Availability saved"
            await loadZones()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
