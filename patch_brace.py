import re

with open("pegasus/apps/retailer-app-ios/retailerapp/reatilerapp/Screens/ProfileView.swift", "r") as f:
    text = f.read()

text = text.replace("""        } catch {
    }

    private func loadStats() async {""", """        } catch {
            print("Failed to load profile: \\(error)")
        }
    }

    private func loadStats() async {""")

with open("pegasus/apps/retailer-app-ios/retailerapp/reatilerapp/Screens/ProfileView.swift", "w") as f:
    f.write(text)

