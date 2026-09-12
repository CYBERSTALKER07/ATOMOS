import SwiftUI

struct CoverageView: View {
    @State private var coverage: WarehouseCoverageResponse?
    @State private var factory: WarehouseSupplyFactoryResponse?
    @State private var factoryError: String?
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                ProgressView().frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") { load() }
                }
            } else {
                List {
                    Section("Coverage") {
                        LabeledContent("Mode", value: coverageModeLabel(coverage?.mode))
                        if let country = coverage?.countryCode, !country.isEmpty {
                            LabeledContent("Country", value: country)
                        }
                        Text("Pins and cities are set by the supplier. This warehouse cannot re-pin.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    Section("Cities") {
                        if let cities = coverage?.cities, !cities.isEmpty {
                            ForEach(cities) { city in
                                Text(city.name)
                            }
                        } else {
                            Text("Closest same-country matching (no city cells).")
                                .foregroundStyle(.secondary)
                        }
                    }
                    Section("Pins") {
                        if let pins = coverage?.pins, !pins.isEmpty {
                            ForEach(pins) { pin in
                                LabeledContent(pin.targetType, value: pin.targetId)
                            }
                        } else {
                            Text("No supplier pins on this warehouse.")
                                .foregroundStyle(.secondary)
                        }
                    }
                    Section("Nearest factory") {
                        if let factory, !factory.factoryId.isEmpty {
                            LabeledContent("Factory", value: factory.factoryId)
                            LabeledContent("Source", value: factory.source)
                            if !factory.countryCode.isEmpty {
                                LabeledContent("Country", value: factory.countryCode)
                            }
                        } else {
                            Text(factoryError ?? "No same-country factory assigned.")
                                .foregroundStyle(.secondary)
                        }
                    }
                }
            }
        }
        .navigationTitle("Coverage and supply")
        .task { load() }
    }

    private func coverageModeLabel(_ mode: String?) -> String {
        switch (mode ?? "").uppercased() {
        case "PINNED": "Pinned"
        case "CITY_CELLS": "City cells"
        default: "Closest in country"
        }
    }

    private func load() {
        loading = true
        error = nil
        factoryError = nil
        Task {
            do {
                coverage = try await WarehouseService.opsCoverage()
            } catch {
                self.error = error.localizedDescription
                loading = false
                return
            }
            do {
                factory = try await WarehouseService.opsSupplyFactory()
            } catch {
                factoryError = error.localizedDescription
            }
            loading = false
        }
    }
}
