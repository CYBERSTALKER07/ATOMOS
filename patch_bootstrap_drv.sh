ed pegasusX/apps/backend-go/bootstrap/bootstrap.go << 'ED'
/var factoryRepo factory.Repository/
i
	var driverRepo driver.Repository
.
/factoryRepo = factory.NewSpannerRepository(spannerClient)/
i
		driverRepo = driver.NewSpannerRepository(spannerClient)
.
/factoryRepo = factory.NewInMemoryRepository()/
i
		driverRepo = driver.NewInMemoryRepository()
.
/driver.ServiceConfig{}/
c
        driverSvc := driver.NewService(driver.ServiceConfig{
            Repo: driverRepo,
        })
.
w
ED
