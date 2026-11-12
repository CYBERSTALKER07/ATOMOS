import SwiftUI

struct LaborCapacityView: View {
    @State private var vm = LaborCapacityViewModel()

    var body: some View {
        @Bindable var vm = vm
        Form {
            Section {
                Text("Zone delivery capacity and driver reliability scores.")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                TextField("Date (YYYY-MM-DD)", text: $vm.date)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                Button("Refresh zones") { Task { await vm.loadZones() } }
                    .disabled(vm.loading)
                    .frame(minHeight: 44)
                    .contentShape(Rectangle())
            }

            if let error = vm.error {
                Section { Text(error).foregroundStyle(.red) }
            }
            if let statusMessage = vm.statusMessage {
                Section { Text(statusMessage).foregroundStyle(.secondary) }
            }

            Section("Zones") {
                if vm.loading {
                    ProgressView()
                } else if vm.zones.isEmpty {
                    Text("No zone capacity rows. Workers populate ZoneCapacity after availability is set.")
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(vm.zones) { z in
                        let util = z.totalCapacity > 0 ? (z.usedCapacity / z.totalCapacity) * 100 : 0
                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                            Text(z.zoneH3).font(.headline).fontDesign(.monospaced)
                            Text(String(format: "Total %.1f · Used %.1f · %.0f%%", z.totalCapacity, z.usedCapacity, util))
                            if !z.date.isEmpty {
                                Text(z.date).font(.caption).foregroundStyle(.secondary)
                            }
                        }
                        .padding(.vertical, 2)
                        .frame(minHeight: 44)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Zone \(z.zoneH3), total \(String(format: "%.1f", z.totalCapacity)), used \(String(format: "%.1f", z.usedCapacity)), \(String(format: "%.0f%%", util)) utilized")
                    }
                }
            }

            Section("Driver score & availability") {
                TextField("Driver ID", text: $vm.driverId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                Button("Load score") { Task { await vm.loadScore() } }
                    .frame(minHeight: 44)
                    .contentShape(Rectangle())
                if let score = vm.score {
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
                TextField("Hours", text: $vm.availHours)
                    .keyboardType(.decimalPad)
                TextField("Status (AVAILABLE / LIMITED / OFF)", text: $vm.availStatus)
                    .textInputAutocapitalization(.characters)
                    .autocorrectionDisabled()
                TextField("Zone H3 (optional)", text: $vm.zoneH3)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                Button(vm.saving ? "Saving…" : "Save availability") {
                    Task { await vm.saveAvailability() }
                }
                .disabled(vm.saving)
                .frame(minHeight: 44)
                .contentShape(Rectangle())
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .background(LabTheme.background)
        .navigationTitle("Labor capacity")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    Task { await vm.loadZones() }
                }
                .frame(minWidth: 44, minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Refresh labor capacity")
            }
        }
        .task { await vm.loadZones() }
        .refreshable { await vm.loadZones() }
    }
}
