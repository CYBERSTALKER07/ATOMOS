import re
with open('the-lab-monorepo/apps/driverappios/driverappios.xcodeproj/project.pbxproj', 'r') as f:
    text = f.read()

text = text.replace('GENERATE_INFOPLIST_FILE = YES;', 'GENERATE_INFOPLIST_FILE = NO;')

with open('the-lab-monorepo/apps/driverappios/driverappios.xcodeproj/project.pbxproj', 'w') as f:
    f.write(text)
