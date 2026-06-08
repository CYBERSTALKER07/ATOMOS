with open("service.go", "r") as f:
    code = f.read()

# Revert s.factoryIDID back to s.factoryID (Wait, no, I just want to replace all s.factoryIDID and s.factoryID to s.factoryNodeID)
code = code.replace('s.factoryIDID', 's.factoryNodeID')
code = code.replace('s.factoryID', 's.factoryNodeID')

with open("service.go", "w") as f:
    f.write(code)
print("Updated service.go factory ID names")
