import re
with open('the-lab-monorepo/apps/driverappios/driverappios.xcodeproj/project.pbxproj', 'r') as f:
    text = f.read()

text = text.replace('INFOPLIST_FILE = driverappios/Info.plist;', 'INFOPLIST_FILE = driverappios/Custom-Info.plist;')
text = text.replace('GENERATE_INFOPLIST_FILE = NO;', 'GENERATE_INFOPLIST_FILE = YES;')

with open('the-lab-monorepo/apps/driverappios/driverappios.xcodeproj/project.pbxproj', 'w') as f:
    f.write(text)
