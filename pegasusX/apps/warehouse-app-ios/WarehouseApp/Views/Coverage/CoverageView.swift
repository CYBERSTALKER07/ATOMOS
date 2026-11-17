import SwiftUI

struct CoverageView: View {
    @State private var vm = CoverageViewModel()

    var body: some View {
        Group {
            if vm.loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.light()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            } else {
                List {
                    Section("Coverage") {
                        LabeledContent("Mode", value: vm.coverageModeLabel(vm.coverage?.mode))
                        if let country = vm.coverage?.countryCode, !country.isEmpty {
                            LabeledContent("Country", value: country)
                        }
                        Text("Pins and cities are set by the supplier. This warehouse cannot re-pin.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    Section("Cities") {
                        if let cities = vm.coverage?.cities, !cities.isEmpty {
                            ForEach(cities) { city in
                                Text(city.name)
                                    .frame(minHeight: 44, alignment: .leading)
                            }
                        } else {
                            Text("Closest same-country matching (no city cells).")
                                .foregroundStyle(.secondary)
                        }
                    }
                    Section("Pins") {
                        if let pins = vm.coverage?.pins, !pins.isEmpty {
                            ForEach(pins) { pin in
                                LabeledContent(pin.targetType, value: pin.targetId)
                                    .frame(minHeight: 44)
                                    .accessibilityElement(children: .combine)
                            }
                        } else {
                            Text("No supplier pins on this warehouse.")
                                .foregroundStyle(.secondary)
                        }
                    }
                    Section("Nearest factory") {
                        if let factory = vm.factory, !factory.factoryId.isEmpty {
                            LabeledContent("Factory", value: factory.factoryId)
                            LabeledContent("Source", value: factory.source)
                            if !factory.countryCode.isEmpty {
                                LabeledContent("Country", value: factory.countryCode)
                            }
                        } else {
                            Text(vm.factoryError ?? "No same-country factory assigned.")
                                .foregroundStyle(.secondary)
                        }
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("Coverage and supply")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh coverage and supply")
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }
}
