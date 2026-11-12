import SwiftUI

public struct DesktopKeyboardCheatsheetView: View {
    @Binding var isPresented: Bool
    @State private var searchText: String = ""

    public init(isPresented: Binding<Bool>) {
        self._isPresented = isPresented
    }

    public struct ShortcutEntry: Identifiable {
        public let id = UUID()
        public let keyCombination: [String]
        public let description: String
        public let category: String
    }

    private let shortcuts: [ShortcutEntry] = [
        // Navigation
        ShortcutEntry(keyCombination: ["⌘", "1"], description: "Navigate to Overview NOC dashboard", category: "Navigation"),
        ShortcutEntry(keyCombination: ["⌘", "2"], description: "Navigate to Multi-Cell Infrastructure", category: "Navigation"),
        ShortcutEntry(keyCombination: ["⌘", "3"], description: "Navigate to FinOps & Billing Ledger", category: "Navigation"),
        ShortcutEntry(keyCombination: ["⌘", "4"], description: "Navigate to Live Telemetry Stream", category: "Navigation"),
        ShortcutEntry(keyCombination: ["⌘", "5"], description: "Navigate to Operational Scorecards", category: "Navigation"),

        // Global Operations
        ShortcutEntry(keyCombination: ["⌘", "R"], description: "Refresh all multi-cell telemetry and metrics", category: "Global Actions"),
        ShortcutEntry(keyCombination: ["⌘", "/"], description: "Toggle this Keyboard Shortcuts Cheatsheet", category: "Global Actions"),
        ShortcutEntry(keyCombination: ["⇧", "⌘", "E"], description: "Export Tender Settlement Waterfall CSV", category: "Global Actions"),
        ShortcutEntry(keyCombination: ["⇧", "⌘", "D"], description: "Export JSON Diagnostic Support Bundle", category: "Global Actions"),

        // Telemetry Stream Controls
        ShortcutEntry(keyCombination: ["Space"], description: "Pause / Resume incoming WebSocket event ingestion", category: "Telemetry & Logs"),
        ShortcutEntry(keyCombination: ["Esc"], description: "Close modal / Dismiss active inspector drawer", category: "Telemetry & Logs")
    ]

    private var filteredCategories: [String] {
        let categories = Array(Set(shortcuts.map(\.category))).sorted()
        if searchText.isEmpty { return categories }
        return categories.filter { cat in
            shortcuts.contains { $0.category == cat &&
                ($0.description.localizedCaseInsensitiveContains(searchText) ||
                 $0.keyCombination.joined().localizedCaseInsensitiveContains(searchText))
            }
        }
    }

    public var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                HStack(spacing: 8) {
                    Image(systemName: "keyboard")
                        .font(.title2)
                        .foregroundStyle(Color.accentColor)
                    VStack(alignment: .leading, spacing: 2) {
                        Text("Pegasus Control Tower")
                            .font(.headline)
                            .fontWeight(.bold)
                        Text("Enterprise Keyboard Shortcuts & Navigation HUD")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
                Spacer()
                Button {
                    ControlTowerHaptics.alignment()
                    isPresented = false
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .font(.title3)
                        .foregroundStyle(.secondary)
                        .frame(minWidth: 32, minHeight: 32)
                        .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Close keyboard shortcuts cheatsheet")
                .keyboardShortcut(.cancelAction)
            }
            .padding(.horizontal, 20)
            .padding(.top, 20)
            .padding(.bottom, 12)

            // Search filter
            HStack {
                Image(systemName: "magnifyingglass")
                    .foregroundStyle(.secondary)
                TextField("Search shortcuts by key or function...", text: $searchText)
                    .textFieldStyle(.plain)
                    .accessibilityLabel("Search keyboard shortcuts")
                if !searchText.isEmpty {
                    Button {
                        ControlTowerHaptics.alignment()
                        searchText = ""
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .foregroundStyle(.secondary)
                            .frame(minWidth: 32, minHeight: 32)
                            .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("Clear shortcut search")
                }
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(Color(NSColor.controlBackgroundColor))
            .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
            .padding(.horizontal, 20)
            .padding(.bottom, 14)

            Divider()

            // Shortcuts List by Category
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    ForEach(filteredCategories, id: \.self) { category in
                        VStack(alignment: .leading, spacing: 8) {
                            Text(category.uppercased())
                                .font(.system(.caption2, design: .monospaced).weight(.bold))
                                .tracking(1.0)
                                .foregroundStyle(.secondary)

                            VStack(spacing: 6) {
                                ForEach(shortcuts.filter { $0.category == category &&
                                    (searchText.isEmpty ||
                                     $0.description.localizedCaseInsensitiveContains(searchText) ||
                                     $0.keyCombination.joined().localizedCaseInsensitiveContains(searchText))
                                }) { item in
                                    HStack {
                                        Text(item.description)
                                            .font(.subheadline)
                                        Spacer()
                                        HStack(spacing: 4) {
                                            ForEach(item.keyCombination, id: \.self) { key in
                                                Text(key)
                                                    .font(.system(.caption, design: .monospaced).weight(.bold))
                                                    .padding(.horizontal, 7)
                                                    .padding(.vertical, 3)
                                                    .background(Color(NSColor.controlBackgroundColor))
                                                    .overlay(
                                                        RoundedRectangle(cornerRadius: 5, style: .continuous)
                                                            .stroke(Color.primary.opacity(0.15), lineWidth: 1)
                                                    )
                                                    .clipShape(RoundedRectangle(cornerRadius: 5, style: .continuous))
                                            }
                                        }
                                    }
                                    .padding(.horizontal, 12)
                                    .padding(.vertical, 8)
                                    .background(Color.primary.opacity(0.02))
                                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                                    .accessibilityElement(children: .combine)
                                    .accessibilityLabel("\(item.description): \(item.keyCombination.joined(separator: " plus "))")
                                }
                            }
                        }
                    }
                }
                .padding(20)
            }

            Divider()

            // Footer
            HStack {
                HStack(spacing: 6) {
                    Circle()
                        .fill(Color.green)
                        .frame(width: 6, height: 6)
                    Text("Pegasus Monochrome Desktop Standard")
                        .font(.system(.caption2, design: .monospaced))
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("Done") {
                    ControlTowerHaptics.alignment()
                    isPresented = false
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.regular)
                .accessibilityLabel("Done")
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 14)
            .background(Color(NSColor.windowBackgroundColor))
        }
        .frame(width: 540, height: 480)
    }
}
