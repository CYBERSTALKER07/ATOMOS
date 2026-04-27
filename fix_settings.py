import sys

with open("the-lab-monorepo/apps/backend-go/settings/empathy_handlers.go", "r") as f:
    lines = f.readlines()

new_lines = []
for i, line in enumerate(lines):
    new_lines.append(line)
    if "log.Printf(\"[EMPATHY ENGINE] %s -> GlobalAutoOrder = %v\", retailerID, req.Enabled)" in line:
        new_lines.insert(len(new_lines)-1, "\tif s.Cache \!= nil {\n\t\ts.Cache.InvalidatePrefix(r.Context(), cache.PrefixSettings+retailerID+\":\")\n\t}\n")
    elif "log.Printf(\"[EMPATHY ENGINE] %s overriding supplier %s -> %v\", retailerID, supplierID, req.Enabled)" in line:
        new_lines.insert(len(new_lines)-1, "\tif s.Cache \!= nil {\n\t\ts.Cache.InvalidatePrefix(r.Context(), cache.PrefixSettings+retailerID+\":\")\n\t}\n")
    elif "log.Printf(\"[EMPATHY ENGINE] %s overriding category %s -> %v\", retailerID, categoryID, req.Enabled)" in line:
        new_lines.insert(len(new_lines)-1, "\tif s.Cache \!= nil {\n\t\ts.Cache.InvalidatePrefix(r.Context(), cache.PrefixSettings+retailerID+\":\")\n\t}\n")
    elif "log.Printf(\"[EMPATHY ENGINE] %s overriding product %s -> %v\", retailerID, productID, req.Enabled)" in line:
        new_lines.insert(len(new_lines)-1, "\tif s.Cache \!= nil {\n\t\ts.Cache.InvalidatePrefix(r.Context(), cache.PrefixSettings+retailerID+\":\")\n\t}\n")
    elif "log.Printf(\"[EMPATHY ENGINE] %s overriding variant %s -> %v\", retailerID, skuID, req.Enabled)" in line:
        new_lines.insert(len(new_lines)-1, "\tif s.Cache \!= nil {\n\t\ts.Cache.InvalidatePrefix(r.Context(), cache.PrefixSettings+retailerID+\":\")\n\t}\n")

with open("the-lab-monorepo/apps/backend-go/settings/empathy_handlers.go", "w") as f:
    f.writelines(new_lines)
