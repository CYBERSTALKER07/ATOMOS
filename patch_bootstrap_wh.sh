ed pegasusX/apps/backend-go/bootstrap/bootstrap.go << 'ED'
/warehouseSvc := warehouse.NewService/
i
	var warehouseRepo warehouse.Repository
	if spannerClient != nil {
		warehouseRepo = warehouse.NewSpannerRepository(spannerClient)
	} else {
		warehouseRepo = warehouse.NewInMemoryRepository()
	}

.
/SupplierID: supplierSeed.SupplierID/
i
		Repo:       warehouseRepo,
.
w
ED
