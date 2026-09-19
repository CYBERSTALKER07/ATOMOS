import SwiftUI

struct ColdChainView: View {
    @State private var manifestId = ""
    @State private var sensorId = ""
    @State private var tempC = ""
    @State private var minC = ""
    @State private var maxC = ""
    @State private var readings: [TemperatureReading] = []
    @State private var enabled = true
    @State private var loading = false
    @State private var posting = false
    @State private var error: String?
    @State private var statusMessage: String?

    var body: some View {
        NavigationStack {
            Group {
                if !enabled {
                    ContentUnavailableView {
                        Label("Cold chain disabled", systemImage: "thermometer.medium.slash")
                    } description: {
                        Text("Set WMS_COLD_CHAIN_ENABLED=true on the API to enable temperature ingest.")
                    }
                } else {
                    Form {
                        Section {
                            Text("Manifest temperature readings — excursions quarantine lots and raise system breaches.")
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                            TextField("Manifest ID", text: $manifestId)
                                .textInputAutocapitalization(.never)
                                .autocorrectionDisabled()
                            Button(loading ? "Loading…" : "Load readings") {
                                Task { await load() }
                            }
                            .disabled(loading)
                        }

                        Section("Record reading") {
                            TextField("Sensor ID", text: $sensorId)
                                .textInputAutocapitalization(.never)
                                .autocorrectionDisabled()
                            TextField("Temp °C", text: $tempC)
                                .keyboardType(.decimalPad)
                            TextField("Min °C (optional)", text: $minC)
                                .keyboardType(.decimalPad)
                            TextField("Max °C (optional)", text: $maxC)
                                .keyboardType(.decimalPad)
                            Button(posting ? "Recording…" : "Record reading") {
                                Task { await ingest() }
                            }
                            .disabled(posting)
                        }

                        if let error {
                            Section {
                                Text(error).foregroundStyle(.red)
                            }
                        }
                        if let statusMessage {
                            Section {
                                Text(statusMessage).foregroundStyle(.secondary)
                            }
                        }

                        Section("Readings") {
                            if !loading && readings.isEmpty {
                                Text("No readings — load a manifest or record the first sample.")
                                    .foregroundStyle(.secondary)
                            }
                            ForEach(readings) { row in
                                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                    HStack {
                                        Text(String(format: "%.2f °C", row.tempC))
                                            .font(.headline)
                                        Spacer()
                                        Text(row.excursion ? "EXCURSION" : "OK")
                                            .font(.caption)
                                            .foregroundStyle(row.excursion ? Color.red : Color.secondary)
                                    }
                                    Text(row.recordedAt)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                    Text(bandLabel(for: row))
                                        .font(.caption2)
                                        .foregroundStyle(.secondary)
                                }
                                .padding(.vertical, 2)
                            }
                        }
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Cold chain")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                        Task { await load() }
                    }
                }
            }
        }
    }

    private func load() async {
        let mid = manifestId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !mid.isEmpty else {
            error = "Manifest ID required"
            return
        }
        loading = true
        error = nil
        statusMessage = nil
        defer { loading = false }
        do {
            let resp = try await WarehouseService.temperatureReadings(manifestId: mid)
            enabled = true
            readings = resp.readings
        } catch let APIError.httpError(code) where code == 409 {
            enabled = false
            readings = []
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func ingest() async {
        let mid = manifestId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let temp = Double(tempC.trimmingCharacters(in: .whitespacesAndNewlines)), !mid.isEmpty else {
            error = "Manifest ID and temperature required"
            return
        }
        let minVal = Double(minC.trimmingCharacters(in: .whitespacesAndNewlines))
        let maxVal = Double(maxC.trimmingCharacters(in: .whitespacesAndNewlines))
        posting = true
        error = nil
        statusMessage = nil
        defer { posting = false }
        do {
            let body = TemperatureReadingIngestRequest(
                manifestId: mid,
                tempC: temp,
                sensorId: sensorId.trimmingCharacters(in: .whitespacesAndNewlines),
                minC: (minVal != nil && maxVal != nil) ? minVal : nil,
                maxC: (minVal != nil && maxVal != nil) ? maxVal : nil
            )
            _ = try await WarehouseService.ingestTemperatureReading(body)
            statusMessage = "Reading recorded"
            tempC = ""
            await load()
        } catch let APIError.httpError(code) where code == 409 {
            enabled = false
            error = "Cold chain disabled on this environment"
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func bandLabel(for row: TemperatureReading) -> String {
        let band: String
        if let minC = row.minC, let maxC = row.maxC {
            band = "\(minC)…\(maxC)"
        } else {
            band = "—"
        }
        return "Band \(band) · Sensor \(row.sensorId ?? "—")"
    }
}
