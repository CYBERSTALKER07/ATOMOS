with open("the-lab-monorepo/apps/driverappios/driverappios/Views/FleetMapView.swift", "r") as f:
    text = f.read()

text = text.replace(
    ".background(isCameraLocked ? LabTheme.fg : .ultraThinMaterial)",
    ".background(isCameraLocked ? AnyShapeStyle(LabTheme.fg) : AnyShapeStyle(.ultraThinMaterial))"
)

with open("the-lab-monorepo/apps/driverappios/driverappios/Views/FleetMapView.swift", "w") as f:
    f.write(text)
