import re

with open("apps/backend-go/driver/repository_crud.go", "r") as f:
    content = f.read()

# Replace spanner.UpdateStruct("Vehicles", v) with UpdateMap
pattern = re.compile(r'func \(r \*SpannerRepository\) UpdateVehicle\(ctx context\.Context, v Vehicle\) error \{\n\t\.\.\.')
# wait I need to see the function first.
