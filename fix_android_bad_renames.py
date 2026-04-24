import os
import glob

def fix_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    # Revert erroneous replacements done by an overly aggressive string replacement script
    content = content.replace('GlobalPaynt', 'Payment')
    content = content.replace('GlobalPaynts', 'Payments')
    content = content.replace('global_paynt', 'payment')
    content = content.replace('cashable', 'clickable')
    content = content.replace('onCash', 'onClick')
    
    with open(filepath, 'w') as f:
        f.write(content)

app_dir = 'the-lab-monorepo/apps/retailer-app-android/app/src/main/java/com/thelab/retailer'
for root, _, files in os.walk(app_dir):
    for file in files:
        if file.endswith('.kt'):
            fix_file(os.path.join(root, file))

print("Fixed broken strings in Android app\!")
