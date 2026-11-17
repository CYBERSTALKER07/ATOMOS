import SwiftUI

struct ResponsiveGridContentWrapper<Content: View>: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @ViewBuilder let content: () -> Content

    var body: some View {
        if horizontalSizeClass == .regular {
            ScrollView {
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 340), spacing: 16)], spacing: 16) {
                    content()
                        .padding()
                        .background(Color(UIColor.secondarySystemGroupedBackground))
                        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                }
                .padding()
            }
            .background(Color(UIColor.systemGroupedBackground))
        } else {
            List {
                content()
            }
            .listStyle(.insetGrouped)
        }
    }
}
