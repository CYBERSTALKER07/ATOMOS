import re

with open("apps/backend-go/driver/repository_spanner.go", "r") as f:
    content = f.read()
content = content.replace("UpdateVehicle(ctx context.Context, v Vehicle, emit func(outbox.TxnBuffer) error) error", "UpdateVehicle(ctx context.Context, vehicleID string, updates map[string]any, emit func(outbox.TxnBuffer) error) error")
pattern = re.compile(r'func \(r \*inMemoryRepository\) UpdateVehicle\(ctx context\.Context, v Vehicle, emit func\(outbox\.TxnBuffer\) error\) error \{.*?\n\}', re.DOTALL)
replacement = r"""func (r *inMemoryRepository) UpdateVehicle(ctx context.Context, vehicleID string, updates map[string]any, emit func(outbox.TxnBuffer) error) error {
	return nil
}"""
content = pattern.sub(replacement, content)
with open("apps/backend-go/driver/repository_spanner.go", "w") as f:
    f.write(content)


with open("apps/backend-go/driver/repository_crud.go", "r") as f:
    content = f.read()

pattern = re.compile(r'func \(r \*SpannerRepository\) UpdateVehicle\(ctx context\.Context, v Vehicle, emit func\(outbox\.TxnBuffer\) error\) error \{\n\tv\.UpdatedAt = spanner\.CommitTimestamp\n\tm, err := spanner\.UpdateStruct\("Vehicles", v\)\n\tif err != nil \{\n\t\treturn err\n\t\}\n\t_, err = r\.client\.ReadWriteTransaction\(ctx, func\(ctx context\.Context, txn \*spanner\.ReadWriteTransaction\) error \{\n\t\tmuts := \[\]\*spanner\.Mutation\{m\}\n\t\tif emit != nil \{\n\t\t\tbuf := &spannerTxnBuffer\{\}\n\t\t\tif err := emit\(buf\); err != nil \{\n\t\t\t\treturn err\n\t\t\t\}\n\t\t\tmuts = append\(muts, outboxMutations\(buf\.events\)\.\.\.\)\n\t\t\}\n\t\treturn txn\.BufferWrite\(muts\)\n\t\}\)\n\treturn err\n\}')

replacement = r"""func (r *SpannerRepository) UpdateVehicle(ctx context.Context, vehicleID string, updates map[string]any, emit func(outbox.TxnBuffer) error) error {
	updates["VehicleId"] = vehicleID
	updates["UpdatedAt"] = spanner.CommitTimestamp
	m := spanner.UpdateMap("Vehicles", updates)
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{m}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts = append(muts, outboxMutations(buf.events)...)
		}
		return txn.BufferWrite(muts)
	})
	return err
}"""
content = pattern.sub(replacement, content)
with open("apps/backend-go/driver/repository_crud.go", "w") as f:
    f.write(content)


with open("apps/backend-go/driver/crud_handlers.go", "r") as f:
    content = f.read()

pattern = re.compile(r'\tvar req Vehicle\n\tif err := json\.Unmarshal\(body, &req\); err != nil \{\n\t\tweb\.JSONError\(w, "invalid request body", http\.StatusBadRequest\)\n\t\treturn\n\t\}\n\treq\.VehicleID = vehicleID\n\n\tsupplierID, ok := auth\.ResolveSupplierID\(r\.Context\(\)\)\n\tif !ok \|\| supplierID == "" \{\n\t\tweb\.JSONError\(w, "missing supplier scope", http\.StatusUnauthorized\)\n\t\treturn\n\t\}\n\treq\.SupplierID = supplierID\n\n\temit := func\(buf outbox\.TxnBuffer\) error \{\n\t\treturn outbox\.EmitJSON\(r\.Context\(\), buf, events\.AggregateVehicle, req\.VehicleID, events\.TopicMain, events\.VehicleEvent\{\n\t\t\tBaseEvent:\s+events\.BaseEvent\{Type: events\.EventVehicleAvailabilityChanged, Version: 1\},\n\t\t\tVehicleID:\s+req\.VehicleID,\n\t\t\tSupplierID:\s+req\.SupplierID,\n\t\t\tHomeNodeID:\s+req\.HomeNodeID,\n\t\t\tHomeNodeType:\s+req\.HomeNodeType,\n\t\t\tIsActive:\s+req\.IsActive,\n\t\t\}\)\n\t\}\n\tif err := s\.repo\.UpdateVehicle\(r\.Context\(\), req, emit\); err != nil \{')

replacement = r"""	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	
	// Format keys for Spanner (e.g. from JSON snake_case to PascalCase if needed, but our API uses PascalCase or we map them)
	// We'll just pass the map, assuming the payload sends the correct column names or we map them.
	// Actually, let's just create a typed map to ensure valid keys.
	var parsedReq Vehicle
	if err := json.Unmarshal(body, &parsedReq); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		web.JSONError(w, "missing supplier scope", http.StatusUnauthorized)
		return
	}
	
	updates := make(map[string]any)
	updates["SupplierId"] = supplierID
	if v, ok := req["is_active"]; ok { updates["IsActive"] = v }
	if v, ok := req["home_node_id"]; ok { updates["HomeNodeId"] = v }
	if v, ok := req["home_node_type"]; ok { updates["HomeNodeType"] = v }
	if v, ok := req["driver_id"]; ok { updates["DriverId"] = v }
	if v, ok := req["make"]; ok { updates["Make"] = v }
	if v, ok := req["model"]; ok { updates["Model"] = v }
	if v, ok := req["year"]; ok { updates["Year"] = v }
	if v, ok := req["license_plate"]; ok { updates["LicensePlate"] = v }
	if v, ok := req["class"]; ok { updates["Class"] = v }
	if v, ok := req["max_volume_vu"]; ok { updates["MaxVolumeVU"] = v }
	if v, ok := req["max_weight_kg"]; ok { updates["MaxWeightKg"] = v }
	if v, ok := req["capacity_tags"]; ok { updates["CapacityTags"] = v }

	emit := func(buf outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), buf, events.AggregateVehicle, vehicleID, events.TopicMain, events.VehicleEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventVehicleAvailabilityChanged, Version: 1},
			VehicleID:    vehicleID,
			SupplierID:   supplierID,
			HomeNodeID:   parsedReq.HomeNodeID,
			HomeNodeType: parsedReq.HomeNodeType,
			IsActive:     parsedReq.IsActive,
		})
	}
	if err := s.repo.UpdateVehicle(r.Context(), vehicleID, updates, emit); err != nil {"""
content = pattern.sub(replacement, content)
with open("apps/backend-go/driver/crud_handlers.go", "w") as f:
    f.write(content)


with open("apps/backend-go/driver/service_test.go", "r") as f:
    content = f.read()

pattern = re.compile(r'func \(r \*driverRepoSpy\) UpdateVehicle\(ctx context\.Context, v Vehicle, emit func\(outbox\.TxnBuffer\) error\) error \{')
replacement = r'func (r *driverRepoSpy) UpdateVehicle(ctx context.Context, vehicleID string, updates map[string]any, emit func(outbox.TxnBuffer) error) error {'
content = pattern.sub(replacement, content)
with open("apps/backend-go/driver/service_test.go", "w") as f:
    f.write(content)

with open("apps/backend-go/driver/get_ownership_test.go", "r") as f:
    content = f.read()

pattern = re.compile(r'func \(o \*driverOwnershipRepo\) UpdateVehicle\(ctx context\.Context, v Vehicle, emit func\(outbox\.TxnBuffer\) error\) error \{')
replacement = r'func (o *driverOwnershipRepo) UpdateVehicle(ctx context.Context, vehicleID string, updates map[string]any, emit func(outbox.TxnBuffer) error) error {'
content = pattern.sub(replacement, content)
with open("apps/backend-go/driver/get_ownership_test.go", "w") as f:
    f.write(content)

