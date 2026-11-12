import SwiftUI

struct LocationSetupView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var vm = LocationSetupViewModel()

    private var hasAssignedFactory: Bool { tokenStore.hasAssignedFactory }

    var body: some View {
        @Bindable var vm = vm

        Group {
            if vm.loading {
                ProgressView("Loading…")
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                Form {
                    Section {
                        Text(
                            hasAssignedFactory
                                ? "Confirm or update your facility address. Changes sync with supply routing and loading bay operations."
                                : "Name your factory and set the facility address to start operations."
                        )
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                    }
                    if !hasAssignedFactory {
                        Section("factory_portal.residual.text.factory_name") {
                            TextField("factory_portal.residual.text.factory_name", text: $vm.factoryName)
                                .frame(minHeight: 44)
                                .accessibilityLabel("Factory name")
                        }
                    } else if !vm.factoryName.isEmpty {
                        Section {
                            Text(vm.factoryName)
                                .accessibilityLabel("Factory name: \(vm.factoryName)")
                        }
                    }
                    Section("Factory address") {
                        AddressLocationField(value: $vm.location, label: "Factory address")
                    }
                    if let error = vm.error {
                        Section {
                            Text(error)
                                .foregroundStyle(.red)
                                .accessibilityLabel("Setup error: \(error)")
                        }
                    }
                    Section {
                        Button(vm.submitting ? "Saving…" : "Complete setup") {
                            Task {
                                await vm.submit(
                                    hasAssignedFactory: hasAssignedFactory,
                                    currentRefreshToken: tokenStore.refreshToken
                                ) { token, refresh, factoryId in
                                    tokenStore.updateTokens(token: token, refresh: refresh, factoryId: factoryId)
                                }
                            }
                        }
                        .frame(minWidth: 44, minHeight: 44)
                        .disabled(vm.submitting)
                        .accessibilityLabel("Complete factory location setup")
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("factory_portal.settings.location.text.factory_location")
        .task {
            await vm.loadExistingIfNeeded(hasAssignedFactory: hasAssignedFactory)
        }
    }
}
