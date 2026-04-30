//
//  OfflineVerifierView.swift
//  driverappios
//

import AVFoundation
import SwiftData
import SwiftUI

struct OfflineVerifierView: View {
    @State private var vm = OfflineVerifierViewModel()
    let modelContext: ModelContext

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: LabTheme.s16) {
                    headerView
                    statusBar
                    stateContent
                }
                .padding(.horizontal, LabTheme.s16)
                .padding(.bottom, 40)
            }
            .background(LabTheme.bg)
            .navigationBarTitleDisplayMode(.inline)
        }
        .onAppear {
            vm.store = OfflineDeliveryStore(modelContext: modelContext)
        }
    }

    // MARK: - Header

    private var headerView: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("OFFLINE VERIFICATION TERMINAL")
                .font(.system(size: 10, weight: .bold, design: .monospaced))
                .foregroundStyle(LabTheme.fgTertiary)

            Text("Hash Manifest Protocol")
                .font(.system(size: 20, weight: .bold))
                .foregroundStyle(LabTheme.fg)
        }
        .padding(.top, LabTheme.s24)
    }

    // MARK: - Status Bar

    private var statusBar: some View {
        HStack {
            Text("Protocol Status")
                .font(.caption.weight(.medium))
                .foregroundStyle(LabTheme.fgSecondary)
            Spacer()
            StatusPill(label: vm.statusLabel, color: vm.statusColor)
        }
        .padding(LabTheme.s16)
        .labCard()
    }

    // MARK: - State Content

    @ViewBuilder
    private var stateContent: some View {
        switch vm.state {
        case .idle:           idleCard
        case .syncing:        syncingCard
        case .ready(let m):   readyCard(m)
        case .scanning:       scanningCard
        case .verified(let o):verifiedCard(o)
        case .fraud(let r):   fraudCard(r)
        case .error(let r):   errorCard(r)
        }
    }

    // MARK: - Idle

    private var idleCard: some View {
        VStack(spacing: 16) {
            Image(systemName: "shield.lefthalf.filled")
                .font(.system(size: 40))
                .foregroundStyle(LabTheme.fg)

            Text("Offline cryptographic verification allows delivery confirmation without network connectivity.")
                .font(.subheadline)
                .foregroundStyle(LabTheme.fgSecondary)
                .multilineTextAlignment(.center)

            Text("Download your route manifest to begin.")
                .font(.subheadline)
                .foregroundStyle(LabTheme.fgSecondary)

            monoButton("Sync Route Manifest") {
                Task { await vm.syncManifest() }
            }
        }
        .padding(LabTheme.s24)
        .labCard()
    }

    // MARK: - Syncing

    private var syncingCard: some View {
        VStack(spacing: 12) {
            ProgressView()
                .controlSize(.large)
                .tint(LabTheme.fg)
            Text("Downloading manifest...")
                .font(.subheadline)
                .foregroundStyle(LabTheme.fgSecondary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 40)
        .labCard()
    }

    // MARK: - Ready

    private func readyCard(_ manifest: RouteManifest) -> some View {
        VStack(alignment: .leading, spacing: 14) {
            Label("Manifest Loaded", systemImage: "checkmark.shield.fill")
                .font(.headline)
                .foregroundStyle(LabTheme.success)

            VStack(alignment: .leading, spacing: 8) {
                infoRow("Driver", manifest.driver_id)
                infoRow("Date", manifest.date)
                infoRow("Orders", "\(manifest.hashes.count)")
                infoRow("Valid", manifest.isValid ? "Yes" : "Expired")
            }

            monoButton("Activate Scanner") { vm.activateScanner() }

            Button {
                Task { await vm.syncManifest() }
            } label: {
                Text("Re-sync Manifest")
                    .font(.subheadline.weight(.bold))
                    .foregroundStyle(LabTheme.fgSecondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 12)
            }
            .buttonStyle(.pressable)
        }
        .padding(LabTheme.s16)
        .labCard()
    }

    // MARK: - Scanning

    private var scanningCard: some View {
        VStack(spacing: 14) {
            QRCameraPreview(onScan: { value in
                vm.handleBarcodeScan(value)
            })
            .frame(height: 300)
            .clipShape(.rect(cornerRadius: LabTheme.cardRadius))

            Text("Point at retailer QR code")
                .font(.subheadline)
                .foregroundStyle(LabTheme.fgSecondary)

            Button {
                vm.cancelScanner()
            } label: {
                Text("Cancel")
                    .font(.subheadline.weight(.bold))
                    .foregroundStyle(LabTheme.destructive)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 12)
            }
            .buttonStyle(.pressable)
        }
    }

    // MARK: - Verified

    private func verifiedCard(_ orderId: String) -> some View {
        VStack(spacing: 14) {
            Image(systemName: "checkmark.circle.fill")
                .font(.system(size: 48))
                .foregroundStyle(LabTheme.success)

            Text("✓ Verified")
                .font(.system(size: 28, weight: .bold))
                .foregroundStyle(LabTheme.success)

            Text(orderId)
                .font(.system(.headline, design: .monospaced))
                .foregroundStyle(LabTheme.fg)

            Text("SHA-256 match confirmed. Delivery queued for sync.")
                .font(.subheadline)
                .foregroundStyle(LabTheme.fgSecondary)
                .multilineTextAlignment(.center)

            monoButton("Next Delivery") { vm.nextDelivery() }
        }
        .padding(LabTheme.s24)
        .labCard()
        .overlay {
            RoundedRectangle(cornerRadius: LabTheme.cardRadius)
                .stroke(LabTheme.success.opacity(0.3), lineWidth: 1)
        }
    }

    // MARK: - Fraud

    private func fraudCard(_ reason: String) -> some View {
        VStack(spacing: 14) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.system(size: 48))
                .foregroundStyle(LabTheme.destructive)

            Text("Fraud Detected")
                .font(.title2.bold())
                .foregroundStyle(LabTheme.destructive)

            Text(reason)
                .font(.system(.subheadline, design: .monospaced))
                .foregroundStyle(LabTheme.destructive)
                .multilineTextAlignment(.center)

            destructiveButton("Reset Terminal") { vm.resetTerminal() }
        }
        .padding(LabTheme.s24)
        .labCard()
        .overlay {
            RoundedRectangle(cornerRadius: LabTheme.cardRadius)
                .stroke(LabTheme.destructive.opacity(0.4), lineWidth: 1.5)
        }
    }

    // MARK: - Error

    private func errorCard(_ reason: String) -> some View {
        VStack(spacing: 14) {
            Image(systemName: "xmark.octagon.fill")
                .font(.system(size: 48))
                .foregroundStyle(LabTheme.destructive)

            Text("System Error")
                .font(.title2.bold())
                .foregroundStyle(LabTheme.destructive)

            Text(reason)
                .font(.subheadline)
                .foregroundStyle(LabTheme.fgSecondary)
                .multilineTextAlignment(.center)

            destructiveButton("Reset Terminal") { vm.resetTerminal() }
        }
        .padding(LabTheme.s24)
        .labCard()
        .overlay {
            RoundedRectangle(cornerRadius: LabTheme.cardRadius)
                .stroke(LabTheme.destructive.opacity(0.4), lineWidth: 1.5)
        }
    }

    // MARK: - Helpers

    private func infoRow(_ label: String, _ value: String) -> some View {
        HStack {
            Text(label)
                .font(.caption.weight(.medium))
                .foregroundStyle(LabTheme.fgSecondary)
            Spacer()
            Text(value)
                .font(.system(.caption, design: .monospaced, weight: .bold))
                .foregroundStyle(LabTheme.fg)
        }
    }

    private func monoButton(_ title: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(title)
                .font(.system(size: 15, weight: .bold))
                .foregroundStyle(LabTheme.buttonFg)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 14)
                .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
        }
        .buttonStyle(.pressable)
    }

    private func destructiveButton(_ title: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(title)
                .font(.system(size: 15, weight: .bold))
                .foregroundStyle(.white)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 14)
                .background(LabTheme.destructive, in: .rect(cornerRadius: LabTheme.buttonRadius))
        }
        .buttonStyle(.pressable)
    }
}

#Preview {
    let config = ModelConfiguration(isStoredInMemoryOnly: true)
    let container = try! ModelContainer(for: OfflineDelivery.self, configurations: config)
    OfflineVerifierView(modelContext: container.mainContext)
}
