with open("the-lab-monorepo/apps/driverappios/driverappios/Services/APIClient.swift", "r") as f:
    lines = f.readlines()

new_lines = []
for i, line in enumerate(lines):
    # line 310 is index 309, 311 is index 310
    if i == 310 and line.strip() == "}":
        continue # skip the extra brace
    new_lines.append(line)

with open("the-lab-monorepo/apps/driverappios/driverappios/Services/APIClient.swift", "w") as f:
    f.writelines(new_lines)
