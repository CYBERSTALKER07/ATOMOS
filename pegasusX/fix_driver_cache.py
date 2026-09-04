import re

with open("apps/backend-go/driver/crud_handlers.go", "r") as f:
    content = f.read()

# I did: sed 's/vehicleCacheKey(req.VehicleID)/vehicleCacheKey(vehicleID)/' which affected CreateVehicle as well.
# In HandleCreateVehicle, it should be req.VehicleID.
pattern = re.compile(r'func \(s \*Service\) HandleCreateVehicle\(.*?\}\n\}', re.DOTALL)
def restore_req(m):
    return m.group(0).replace("vehicleCacheKey(vehicleID)", "vehicleCacheKey(req.VehicleID)").replace("vehiclesListCacheKey(supplierID)", "vehiclesListCacheKey(req.SupplierID)")

content = pattern.sub(restore_req, content)

with open("apps/backend-go/driver/crud_handlers.go", "w") as f:
    f.write(content)
