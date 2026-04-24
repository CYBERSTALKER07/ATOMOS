import os
import glob

def fix_file(filepath):
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()

        new_content = content.replace('GlobalPaynt', 'Payment')
        new_content = new_content.replace('global_paynt', 'payment')
        new_content = new_content.replace('GLOBAL_PAYNT', 'PAYMENT')
        new_content = new_content.replace('cashable', 'clickable')
        new_content = new_content.replace('onCash', 'onClick')
        
        if content \!= new_content:
            with open(filepath, 'w', encoding='utf-8') as f:
                f.write(new_content)
    except Exception as e:
        print(f"Error {filepath}: {e}")

base_dir = 'the-lab-monorepo/apps/backend-go'
for root, _, files in os.walk(base_dir):
    for file in files:
        if file.endswith('.go') or file.endswith('.mod') or file.endswith('Routes') or file.endswith('.sum'):
            fix_file(os.path.join(root, file))

print("Fixed broken strings in backend-go\!")
