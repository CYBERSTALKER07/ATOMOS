import re

with open('full_diff.patch', 'r') as f:
    lines = f.readlines()

backend_lines = []
mobile_lines = []
desktop_lines = []
other_lines = []

current_chunk = []
current_category = "other"

for line in lines:
    if line.startswith("diff --git"):
        # Process previous chunk
        if current_chunk:
            if current_category == "backend":
                backend_lines.extend(current_chunk)
            elif current_category == "mobile":
                mobile_lines.extend(current_chunk)
            elif current_category == "desktop":
                desktop_lines.extend(current_chunk)
            else:
                other_lines.extend(current_chunk)
            
        current_chunk = [line]
        
        # Determine new category
        if "apps/backend-go" in line:
            current_category = "backend"
        elif "android" in line.lower() or "ios" in line.lower() or "swift" in line.lower() or "kt" in line.lower():
            current_category = "mobile"
        elif "portal" in line.lower() or "desktop" in line.lower() or "tauri" in line.lower():
            current_category = "desktop"
        else:
            current_category = "other"
    else:
        current_chunk.append(line)

# Flush last chunk
if current_chunk:
    if current_category == "backend":
        backend_lines.extend(current_chunk)
    elif current_category == "mobile":
        mobile_lines.extend(current_chunk)
    elif current_category == "desktop":
        desktop_lines.extend(current_chunk)
    else:
        other_lines.extend(current_chunk)

with open('backend.patch', 'w') as f: f.writelines(backend_lines)
with open('mobile.patch', 'w') as f: f.writelines(mobile_lines)
with open('desktop.patch', 'w') as f: f.writelines(desktop_lines)
with open('other.patch', 'w') as f: f.writelines(other_lines)

print(f"Backend: {len(backend_lines)} lines")
print(f"Mobile: {len(mobile_lines)} lines")
print(f"Desktop: {len(desktop_lines)} lines")
print(f"Other: {len(other_lines)} lines")
