import SwiftUI

/// A custom full-width bottom sheet that doesn't scale or round the presenting view.
/// Supports drag-to-dismiss and tap-outside-to-dismiss.
struct BottomSheetOverlay<Content: View>: View {
    @Binding var isPresented: Bool
    let snapFraction: CGFloat   // e.g. 0.55 for ~medium
    @ViewBuilder var content: () -> Content

    @State private var dragOffset: CGFloat = 0
    @State private var sheetHeight: CGFloat = 0
    @GestureState private var isDragging = false

    var body: some View {
        GeometryReader { geo in
            let maxHeight = geo.size.height
            let restingHeight = maxHeight * snapFraction

            ZStack(alignment: .bottom) {
                // Scrim — tap to dismiss
                if isPresented {
                    Color.black.opacity(0.25)
                        .ignoresSafeArea()
                        .onTapGesture {
                            withAnimation(AnimationConstants.sheet) {
                                isPresented = false
                            }
                        }
                        .transition(.opacity)
                }

                // Sheet
                if isPresented {
                    VStack(spacing: 0) {
                        // Drag indicator
                        Capsule()
                            .fill(AppTheme.separator)
                            .frame(width: 36, height: 5)
                            .padding(.top, 8)
                            .padding(.bottom, 4)

                        content()
                    }
                    .frame(maxWidth: .infinity)
                    .frame(height: restingHeight, alignment: .top)
                    .background(
                        UnevenRoundedRectangle(
                            topLeadingRadius: AppTheme.radiusXL,
                            topTrailingRadius: AppTheme.radiusXL
                        )
                        .fill(AppTheme.cardBackground)
                        .shadow(color: .black.opacity(0.15), radius: 20, y: -4)
                    )
                    .offset(y: max(dragOffset, 0))
                    .gesture(
                        DragGesture()
                            .updating($isDragging) { _, state, _ in state = true }
                            .onChanged { value in
                                dragOffset = value.translation.height
                            }
                            .onEnded { value in
                                let threshold = restingHeight * 0.3
                                if value.translation.height > threshold ||
                                   value.predictedEndTranslation.height > restingHeight * 0.5 {
                                    withAnimation(AnimationConstants.sheet) {
                                        isPresented = false
                                    }
                                }
                                withAnimation(AnimationConstants.fluid) {
                                    dragOffset = 0
                                }
                            }
                    )
                    .transition(.move(edge: .bottom))
                }
            }
            .animation(AnimationConstants.sheet, value: isPresented)
        }
        .ignoresSafeArea(edges: .bottom)
    }
}
