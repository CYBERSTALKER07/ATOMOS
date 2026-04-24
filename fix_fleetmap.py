import re

with open("the-lab-monorepo/apps/driverappios/driverappios/Views/FleetMapView.swift", "r") as f:
    text = f.read()

# Fix LabTheme.primary
text = text.replace("LabTheme.primary", "LabTheme.fg")
text = text.replace("LabTheme.onPrimary", "LabTheme.buttonFg")

# Fix onMapCameraChange
# from .onMapCameraChange(frequency: .onEnd) { context in ... }
# to .onMapCameraChange(frequency: .onEnd) {
#        if isCameraLocked { isCameraLocked = false }
#    }
old_block = """.onMapCameraChange(frequency: .onEnd) { context in
                if isCameraLocked && \!context.followsUserLocation {
                    isCameraLocked = false
                }
            }"""

new_block = """.onMapCameraChange(frequency: .onEnd) {
                // If the user drags we just unlock
                isCameraLocked = false
            }"""

if old_block in text:
    text = text.replace(old_block, new_block)
else:
    # try regex
    text = re.sub(r'\.onMapCameraChange\(frequency: \.onEnd\) \{ context in\s+if isCameraLocked && \!context\.followsUserLocation \{\s+isCameraLocked = false\s+\}\s+\}', new_block, text)

with open("the-lab-monorepo/apps/driverappios/driverappios/Views/FleetMapView.swift", "w") as f:
    f.write(text)

