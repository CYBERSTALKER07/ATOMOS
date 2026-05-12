import re

with open("pegasus/apps/retailer-app-ios/retailerapp/reatilerapp/Screens/CartView.swift", "r") as f:
    text = f.read()

# The View usually ends before #Preview
patch = """
        }
        .task {
            await cart.sync()
        }
        .onChange(of: cart.items.count) { _ in
            Task {
                await cart.sync()
            }
        }
        .onChange(of: cart.totalItems) { _ in
            Task {
                await cart.sync()
            }
        }
"""

# Try to find where the main body VStack ends. Instead of guessing, I'll just replace emptyCartView definition end, wait no, finding where var body ends.
# I see that it returns a VStack. Let's just find the end of `var body: some View {` block.
# Actually let's use a simpler marker: `    private var emptyCartView: some View {`
# The main VStack ends just before that.

text = text.replace("    private var emptyCartView: some View {", patch + "\n    private var emptyCartView: some View {")

with open("pegasus/apps/retailer-app-ios/retailerapp/reatilerapp/Screens/CartView.swift", "w") as f:
    f.write(text)

