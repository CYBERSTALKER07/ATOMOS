//
//  MapMarkerDetailSheet.swift
//  driverappios
//

import CoreLocation
import SwiftUI

/// Custom auto-dismissing bottom half-sheet shown when tapping a map marker.
/// Slides up with spring animation, auto-hides after 5 seconds, supports drag to dismiss.
struct MapMarkerDetailSheet: View {
    @Environment(\.colorScheme) private var cs

    let mission: Mission
    let driverLocation: CLLocationCoordinate2D?
    let isInRange: Bool
    let onScan: () -> Void
    let onCorrection: () -> Void
    let onDismiss: () -> Void

    @State private var dragOffset: Double = 0
    @State private var appeared = false
    @State private var autoHideWorkItem: DispatchWorkItem?

    private var distance: Double? {
        guard let loc = driverLocation else { return nil }
        return haversineDistance(
            from: loc,
            to: CLLocationCoordinate2D(latitude: mission.target_lat, longitude: mission.target_lng)
        )
    }

    var body: some View {
        VStack(spacing: 0) {
            // Drag indicator
            Capsule()
                .fill(LabTheme.fgTertiary.opacity(0.5))
                .frame(width: 32, height: 4)
                .padding(.top, 12)
                .padding(.bottom, 18)

            VStack(alignment: .leading, spacing: 18) {
                headerSection
                endpointCard
                actionButtons
            }
            .padding(.horizontal, LabTheme.s20)
            .padding(.bottom, LabTheme.s20)
        }
        .background(
            ZStack {
                RoundedRectangle(cornerRadius: LabTheme.cardRadius, style: .continuous)
                    .fill(.ultraThinMaterial)
                RoundedRectangle(cornerRadius: LabTheme.cardRadius, style: .continuous)
                    .fill(LabTheme.card.opacity(0.5))
                RoundedRectangle(cornerRadius: LabTheme.cardRadius, style: .continuous)
                    .stroke(LabTheme.separator, lineWidth: 0.5)
            }
            .shadow(color: .black.opacity(cs == .dark ? 0.65 : 0.12), radius: 40, y: 10)
        )
        .padding(.horizontal, LabTheme.s12)
        .padding(.bottom, bottomInset + LabTheme.s8)
        .offset(y: appeared ? dragOffset : 600)
        .gesture(
            DragGesture()
                .onChanged { value in
                    let translation = max(0, value.translation.height)
                    dragOffset = translation
                    cancelAutoHide()
                }
                .onEnded { value in
                    if value.translation.height > 100 {
                        dismiss()
                    } else {
                        withAnimation(Anim.snappy) { dragOffset = 0 }
                        startAutoHide()
                    }
                }
        )
        .onAppear {
            withAnimation(Anim.sheetReveal) { appeared = true }
            startAutoHide()
        }
        .onDisappear { cancelAutoHide() }
    }

    // MARK: - Header

    private var headerSection: some View {
        HStack(alignment: .top) {
            VStack(alignment: .leading, spacing: 4) {
                Text(mission.order_id)
                    .font(.system(.title2, design: .monospaced, weight: .bold))
                    .foregroundStyle(LabTheme.fg)

                HStack(spacing: 6) {
                    Text(mission.gateway)
                        .font(.system(.footnote, weight: .semibold))
                        .padding(.horizontal, 8)
                        .padding(.vertical, 3)
                        .background(LabTheme.fg.opacity(0.08), in: Capsule())

                    Text(mission.amount.formattedAmount)
                        .font(.system(.footnote, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }
            }

            Spacer()

            Button {
                Haptics.light()
                dismiss()
            } label: {
                Image(systemName: "xmark")
                    .font(.system(.caption, weight: .bold))
                    .foregroundStyle(LabTheme.fgSecondary)
                    .frame(width: 44, height: 44)
                    .background(LabTheme.fg.opacity(0.06), in: Circle())
            }
            .accessibilityLabel("Close")
        }
    }

    // MARK: - Endpoint Card

    private var endpointCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("mobile_driver.ui.delivery_endpoint")
                .font(.system(.caption2, design: .monospaced, weight: .bold))
                .foregroundStyle(LabTheme.fgTertiary)

            Text(String(format: "%.4f, %.4f", mission.target_lat, mission.target_lng))
                .font(.system(.callout, design: .monospaced, weight: .semibold))
                .foregroundStyle(LabTheme.fg)

            HStack(spacing: 8) {
                Circle()
                    .fill(isInRange ? LabTheme.success : LabTheme.warning)
                    .frame(width: 8, height: 8)

                if let dist = distance {
                    Text(formattedDistance(dist))
                        .font(.system(.footnote, design: .monospaced, weight: .bold))
                }

                Text(isInRange ? "Geofence cleared" : "Approaching target")
                    .font(.system(.footnote, weight: .medium))
                    .foregroundStyle(isInRange ? LabTheme.success : LabTheme.warning)
            }
        }
        .padding(LabTheme.s16)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LabTheme.fg.opacity(0.03), in: .rect(cornerRadius: LabTheme.cardRadius - 4))
        .overlay {
            RoundedRectangle(cornerRadius: LabTheme.cardRadius - 4, style: .continuous)
                .stroke(LabTheme.separator, lineWidth: 0.5)
        }
        .accessibilityElement(children: .combine)
    }

    // MARK: - Action Buttons

    private var actionButtons: some View {
        VStack(spacing: 10) {
            Button {
                Haptics.medium()
                if isInRange { onScan() } else { Haptics.warning() }
            } label: {
                HStack(spacing: 8) {
                    Image(systemName: isInRange ? "qrcode.viewfinder" : "location.north.fill")
                        .font(.system(.subheadline, weight: .semibold))
                    Text(isInRange ? "Initiate Proof of Delivery" : "Approach Target for Scan")
                        .font(.system(.subheadline, weight: .bold))
                }
                .foregroundStyle(isInRange ? LabTheme.buttonFg : LabTheme.fgSecondary)
                .frame(maxWidth: .infinity, minHeight: 44)
                .padding(.vertical, 8)
                .background(
                    isInRange ? LabTheme.fg : LabTheme.fg.opacity(0.06),
                    in: .rect(cornerRadius: LabTheme.buttonRadius)
                )
            }
            .buttonStyle(.pressable)
            .disabled(!isInRange)

            Button {
                Haptics.light()
                onCorrection()
            } label: {
                HStack(spacing: 6) {
                    Image(systemName: "pencil.and.list.clipboard")
                        .font(.system(.caption))
                    Text("mobile_driver.ui.delivery_correction")
                        .font(.system(.footnote, weight: .semibold))
                }
                .foregroundStyle(LabTheme.fgSecondary)
                .frame(maxWidth: .infinity, minHeight: 44)
            }
            .buttonStyle(.pressable)
        }
    }

    // MARK: - Auto-hide

    private func startAutoHide() {
        cancelAutoHide()
        let item = DispatchWorkItem {
            dismiss()
        }
        autoHideWorkItem = item
        DispatchQueue.main.asyncAfter(deadline: .now() + 5.0, execute: item)
    }

    private func cancelAutoHide() {
        autoHideWorkItem?.cancel()
        autoHideWorkItem = nil
    }

    private func dismiss() {
        withAnimation(Anim.sheetReveal) { appeared = false }
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.35) {
            onDismiss()
        }
    }

    private var bottomInset: Double {
        let scenes = UIApplication.shared.connectedScenes
        let windowScene = scenes.first as? UIWindowScene
        return Double(windowScene?.windows.first?.safeAreaInsets.bottom ?? 0)
    }
}

#Preview {
    ZStack(alignment: .bottom) {
        Color.gray.opacity(0.2).ignoresSafeArea()
        MapMarkerDetailSheet(
            mission: Mission(order_id: "preview", state: "EN_ROUTE", target_lat: 41.3, target_lng: 69.3, amount: 1000, gateway: "CASH", estimated_arrival_at: nil, route_id: nil, sequence_index: nil),
            driverLocation: CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401),
            isInRange: true,
            onScan: {},
            onCorrection: {},
            onDismiss: {}
        )
    }
}
