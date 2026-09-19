import SwiftUI

/// Collapsible sidebar: 88pt icon rail collapsed, 280pt labeled drawer expanded.
struct CollapsibleSidebar<Item: Hashable, Detail: View>: View {
    let title: String
    @Binding var isExpanded: Bool
    @Binding var selection: Item?
    let groups: [(title: String, items: [CollapsibleSidebarItem<Item>])]
    @ViewBuilder let detail: () -> Detail
    let footer: (() -> AnyView)?

    init(
        title: String,
        isExpanded: Binding<Bool>,
        selection: Binding<Item?>,
        groups: [(title: String, items: [CollapsibleSidebarItem<Item>])],
        @ViewBuilder detail: @escaping () -> Detail,
        footer: (() -> AnyView)? = nil,
    ) {
        self.title = title
        _isExpanded = isExpanded
        _selection = selection
        self.groups = groups
        self.detail = detail
        self.footer = footer
    }

    var body: some View {
        HStack(spacing: 0) {
            sidebar
            detail()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(PegasusMonochromeTheme.background)
        }
        .background(PegasusMonochromeTheme.background)
    }

    private var sidebar: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                if isExpanded {
                    Text(title)
                        .font(.headline)
                        .foregroundStyle(PegasusMonochromeTheme.label)
                        .lineLimit(1)
                    Spacer(minLength: 0)
                }
                Button {
                    withAnimation(.spring(response: 0.35, dampingFraction: 0.8)) {
                        isExpanded.toggle()
                    }
                } label: {
                    Image(systemName: "sidebar.left")
                        .font(.system(size: 20, weight: .medium))
                        .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                        .frame(width: 44, height: 44)
                }
                .buttonStyle(.plain)
            }
            .padding(.horizontal, isExpanded ? 16 : 12)
            .padding(.top, 12)

            ScrollView(showsIndicators: false) {
                VStack(alignment: .leading, spacing: 8) {
                    ForEach(Array(groups.enumerated()), id: \.offset) { _, group in
                        if isExpanded {
                            Text(group.title.uppercased())
                                .font(.caption2.weight(.bold))
                                .foregroundStyle(PegasusMonochromeTheme.tertiaryLabel)
                                .padding(.leading, 12)
                                .padding(.top, 8)
                        }
                        ForEach(group.items) { item in
                            sidebarRow(item)
                        }
                    }
                }
                .padding(.vertical, 8)
            }

            if let footer {
                footer()
            }
        }
        .frame(width: isExpanded ? 280 : 88)
        .background(PegasusMonochromeTheme.card)
        .animation(.spring(response: 0.35, dampingFraction: 0.8), value: isExpanded)
    }

    private func sidebarRow(_ item: CollapsibleSidebarItem<Item>) -> some View {
        let selected = selection == item.tag
        return Button {
            selection = item.tag
        } label: {
            HStack(spacing: 12) {
                Image(systemName: item.icon)
                    .font(.system(size: 18, weight: selected ? .semibold : .regular))
                    .frame(width: 28)
                    .foregroundStyle(selected ? PegasusMonochromeTheme.label : PegasusMonochromeTheme.secondaryLabel)
                if isExpanded {
                    Text(item.label)
                        .font(.subheadline.weight(selected ? .semibold : .regular))
                        .foregroundStyle(selected ? PegasusMonochromeTheme.label : PegasusMonochromeTheme.secondaryLabel)
                        .lineLimit(1)
                    Spacer(minLength: 0)
                }
            }
            .padding(.horizontal, isExpanded ? 16 : 12)
            .padding(.vertical, 10)
            .background(
                RoundedRectangle(cornerRadius: PegasusMonochromeTheme.radiusMD, style: .continuous)
                    .fill(selected ? PegasusMonochromeTheme.secondaryBackground : Color.clear)
            )
        }
        .buttonStyle(.plain)
        .padding(.horizontal, 8)
    }
}

struct CollapsibleSidebarItem<Item: Hashable>: Identifiable {
    let id: String
    let tag: Item
    let label: String
    let icon: String

    init(id: String? = nil, tag: Item, label: String, icon: String) {
        self.id = id ?? "\(label)-\(String(describing: tag))"
        self.tag = tag
        self.label = label
        self.icon = icon
    }
}
