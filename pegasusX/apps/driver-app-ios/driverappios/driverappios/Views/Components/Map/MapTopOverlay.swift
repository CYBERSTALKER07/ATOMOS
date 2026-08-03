import SwiftUI
import MapKit

struct MapTopOverlay: View {
    @Binding var phase: MapPhase
    @Binding var selectedMission: Mission?
    @Binding var isCameraLocked: Bool
    @Binding var userPannedAt: Date?
    @Binding var cameraPosition: MapCameraPosition
    @Binding var zoomFocus: ZoomFocus
    var currentTarget: Mission?
    
    var vm: FleetViewModel
    var isLive: Bool
    var goBack: () -> Void
    var cycleZoom: () -> Void
    
    var body: some View {
        VStack {
            HStack(spacing: 10) {
                Button {
                    Haptics.light()
                    if phase == .previewingOrder {
                        withAnimation(Anim.snappy) { selectedMission = nil; phase = .pickingOrder }
                    } else {
                        goBack()
                    }
                } label: {
                    Image(systemName: phase == .previewingOrder ? "chevron.left" : "xmark")
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(LabTheme.fg)
                        .frame(width: 38, height: 38)
                        .background(.ultraThinMaterial, in: Circle())
                        .overlay(Circle().stroke(LabTheme.separator, lineWidth: 0.5))
                        .shadow(color: .black.opacity(0.08), radius: 8, y: 4)
                }
                .accessibilityLabel(phase == .previewingOrder ? "Back" : "Close map")

                Spacer()

                if phase != .pickingOrder, currentTarget != nil {
                    HStack(spacing: 8) {
                        Button {
                            withAnimation(.easeInOut(duration: MapCameraConfig.cameraAnimationSeconds)) {
                                isCameraLocked = true
                                userPannedAt = nil
                                if let coordinate = vm.displayLocation ?? vm.location {
                                    cameraPosition = .camera(
                                        MapCameraMath.trackingCamera(
                                            coordinate: coordinate,
                                            bearing: vm.displayBearing,
                                            speedMps: vm.displaySpeedMps
                                        )
                                    )
                                } else {
                                    cameraPosition = .userLocation(followsHeading: true, fallback: .automatic)
                                }
                            }
                        } label: {
                            Image(systemName: isCameraLocked ? "location.north.line.fill" : "location.fill")
                                .font(.system(size: 13, weight: .bold))
                                .padding(12)
                                .background(isCameraLocked ? AnyShapeStyle(LabTheme.fg) : AnyShapeStyle(.ultraThinMaterial))
                                .foregroundStyle(isCameraLocked ? LabTheme.buttonFg : LabTheme.fg)
                                .clipShape(Circle())
                                .overlay(Circle().stroke(LabTheme.separator, lineWidth: 0.5))
                                .shadow(color: .black.opacity(0.08), radius: 8, y: 4)
                        }

                        Button {
                            Haptics.light()
                            cycleZoom()
                        } label: {
                            HStack(spacing: 4) {
                                Image(systemName: zoomFocus.icon)
                                    .font(.system(size: 10, weight: .bold))
                                Text(zoomFocus.label)
                                    .font(.system(size: 10, weight: .bold))
                            }
                            .foregroundStyle(LabTheme.fg)
                            .padding(.horizontal, 11)
                            .padding(.vertical, 8)
                            .background(.ultraThinMaterial, in: Capsule())
                            .overlay(Capsule().stroke(LabTheme.separator, lineWidth: 0.5))
                            .shadow(color: .black.opacity(0.08), radius: 8, y: 4)
                        }
                        .accessibilityLabel("Zoom focus: \(zoomFocus.label)")
                        .transition(.fadeScale)
                    }
                }

                if vm.activeMission != nil {
                    TelemetryBadge(isLive: isLive)
                        .transition(.fadeScale)
                }
            }
            .padding(.horizontal, LabTheme.s16)
            .padding(.top, 60)
            .animation(Anim.snappy, value: phase)
            .animation(Anim.snappy, value: zoomFocus)

            if phase == .activeDelivery, let cue = vm.navigationCue {
                NavigationCueBanner(cue: cue)
                    .padding(.horizontal, LabTheme.s16)
                    .padding(.top, 10)
                    .transition(.move(edge: .top).combined(with: .opacity))
            }

            Spacer()
        }
    }
}
