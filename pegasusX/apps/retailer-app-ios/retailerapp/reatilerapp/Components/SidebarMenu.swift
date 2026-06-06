import SwiftUI

struct SidebarMenu: View {
    @Binding var isOpen: Bool
    var onNavigate: ((SidebarDestination) -> Void)?

    @State private var dragOffset: Double = 0
    @State private var appeared = false

    var body: some View {
        ZStack(alignment: .leading) {
            // Dimmed background
            if isOpen {
                Color.black.opacity(0.5)
                    .ignoresSafeArea()
                    .onTapGesture {
                        withAnimation(AnimationConstants.fluid) {
                            isOpen = false
                        }
                    }
                    .transition(.opacity)
            }

            // Sidebar panel
            GeometryReader { geo in
                let menuWidth = min(geo.size.width * 0.82, 340.0)

                HStack(spacing: 0) {
                    VStack(alignment: .leading, spacing: 0) {
                        // Header
                        sidebarHeader
                            .padding(.top, AppTheme.spacingXL)
                            .padding(.horizontal, AppTheme.spacingXL)
                            .padding(.bottom, AppTheme.spacingXL)

                        // Menu Items
                        ScrollView {
                            VStack(alignment: .leading, spacing: AppTheme.spacingXS) {
                                ForEach(Array(SidebarDestination.menuItems.enumerated()), id: \.element) { index, destination in
                                    menuRow(destination)
                                        .opacity(appeared ? 1 : 0)
                                        .offset(x: appeared ? 0 : -20)
                                        .animation(
                                            .spring(response: 0.4, dampingFraction: 0.8).delay(Double(index) * 0.05),
                                            value: appeared
                                        )
                                }
                            }
                            .padding(.horizontal, AppTheme.spacingMD)
                        }
                        .scrollIndicators(.hidden)

                        Spacer()

                        // Logout
                        Rectangle()
                            .fill(AppTheme.separator.opacity(0.3))
                            .frame(height: AppTheme.separatorHeight)
                            .padding(.horizontal, AppTheme.spacingXL)

                        menuRow(.logout)
                            .padding(.horizontal, AppTheme.spacingMD)
                            .padding(.bottom, AppTheme.spacingLG)
                            .padding(.top, AppTheme.spacingSM)

                        // Version
                        Text("Pegasus · v1.0.0")
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                            .padding(.horizontal, AppTheme.spacingXL)
                            .padding(.bottom, AppTheme.spacingMD)
                    }
                    .frame(width: menuWidth)
                    .background {
                        Rectangle()
                            .fill(AppTheme.cardBackground)
                            .ignoresSafeArea()
                    }

                    Spacer(minLength: 0)
                }
                .offset(x: isOpen ? dragOffset : -menuWidth)
                .animation(AnimationConstants.fluid, value: isOpen)
                .gesture(
                    DragGesture()
                        .onChanged { value in
                            if value.translation.width < 0 {
                                dragOffset = value.translation.width
                            }
                        }
                        .onEnded { value in
                            if value.translation.width < -80 {
                                isOpen = false
                            }
                            dragOffset = 0
                        }
                )
            }
        }
        .allowsHitTesting(isOpen)
        .animation(AnimationConstants.fluid, value: isOpen)
        .onAppear {
            if isOpen {
                // Delay slightly so the slide-in finishes before items animate
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.05) {
                    withAnimation {
                        appeared = true
                    }
                }
            }
        }
        .onChange(of: isOpen) {
            if isOpen {
                appeared = false
                withAnimation {
                    appeared = true
                }
            } else {
                appeared = false
            }
        }
    }

    // MARK: - Header

    private var sidebarHeader: some View {
        HStack(spacing: AppTheme.spacingMD) {
            ZStack {
                Circle()
                    .fill(AppTheme.accentGradient)
                    .frame(width: 52, height: 52)
                Text(String((AuthManager.shared.currentUser?.name ?? "U").prefix(1)))
                    .font(.system(.title3, design: .rounded, weight: .bold))
                    .foregroundStyle(.white)
            }

            VStack(alignment: .leading, spacing: 3) {
                Text(AuthManager.shared.currentUser?.name ?? "—")
                    .font(.system(.headline, design: .rounded))
                    .foregroundStyle(AppTheme.textPrimary)
                Text(AuthManager.shared.currentUser?.company ?? "—")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            Spacer()
        }
    }

    // MARK: - Menu Row

    private func menuRow(_ destination: SidebarDestination) -> some View {
        let isLogout = destination == .logout

        return Button {
            Haptics.light()
            onNavigate?(destination)
            withAnimation(AnimationConstants.fluid) {
                isOpen = false
            }
        } label: {
            HStack(spacing: AppTheme.spacingMD) {
                ZStack {
                    RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                        .fill(isLogout ? AppTheme.destructiveSoft.opacity(0.5) : AppTheme.accentSoft.opacity(0.4))
                        .frame(width: 36, height: 36)
                    Image(systemName: destination.icon)
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(isLogout ? AppTheme.destructive : AppTheme.accent)
                }

                Text(destination.title)
                    .font(.system(.body, design: .rounded, weight: .medium))
                    .foregroundStyle(isLogout ? AppTheme.destructive : AppTheme.textPrimary)

                Spacer()

                if !isLogout {
                    Image(systemName: "chevron.right")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(AppTheme.textTertiary.opacity(0.5))
                }
            }
            .padding(.horizontal, AppTheme.spacingMD)
            .padding(.vertical, AppTheme.spacingSM)
            .background {
                RoundedRectangle(cornerRadius: AppTheme.radiusMD)
                    .fill(.clear)
            }
            .contentShape(RoundedRectangle(cornerRadius: AppTheme.radiusMD))
        }
        .buttonStyle(.plain)
    }
}

// MARK: - Sidebar Destination

enum SidebarDestination: String, CaseIterable {
    case dashboard
    case procurement
    case insights
    case autoOrder
    case futureDemand
    case inbox
    case profile
    case settings
    case logout

    /// Menu items shown in the scrollable list (excludes logout)
    static var menuItems: [SidebarDestination] {
        allCases.filter { $0 != .logout }
    }

    var title: String {
        switch self {
        case .dashboard: "Dashboard"
        case .procurement: "Procurement"
        case .insights: "Insights"
        case .autoOrder: "Auto-Order"
        case .futureDemand: "AI Predictions"
        case .inbox: "Inbox"
        case .profile: "Profile"
        case .settings: "Settings"
        case .logout: "Log Out"
        }
    }

    var icon: String {
        switch self {
        case .dashboard: "square.grid.2x2"
        case .procurement: "chart.bar"
        case .insights: "chart.line.uptrend.xyaxis"
        case .autoOrder: "wand.and.stars"
        case .futureDemand: "sparkles"
        case .inbox: "tray"
        case .profile: "person"
        case .settings: "gearshape"
        case .logout: "rectangle.portrait.and.arrow.right"
        }
    }
}

#Preview {
    @Previewable @State var isOpen = true
    SidebarMenu(isOpen: $isOpen)
}
