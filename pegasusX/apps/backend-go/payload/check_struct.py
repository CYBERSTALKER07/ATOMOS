with open("service.go", "r") as f:
    lines = f.readlines()

in_struct = False
for line in lines:
    if line.startswith("type Service struct"):
        in_struct = True
    if in_struct:
        print(line, end="")
        if line.startswith("}"):
            break
