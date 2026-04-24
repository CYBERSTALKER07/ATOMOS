import re
with open('the-lab-monorepo/apps/driverappios/driverappios.xcodeproj/project.pbxproj', 'r') as f:
    text = f.read()

# Specifically look for lines referencing Info.plist inside the PBXBuildFile and PBXResourcesBuildPhase
text = re.sub(r'^[ \t]*[A-Z0-9]+[ \t]+\/\*[ \t]+Info\.plist in Resources[ \t]+\*\/,?\n?', '', text, flags=re.MULTILINE)

with open('the-lab-monorepo/apps/driverappios/driverappios.xcodeproj/project.pbxproj', 'w') as f:
    f.write(text)
