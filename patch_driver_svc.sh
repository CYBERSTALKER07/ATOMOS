sed -i '' 's/type Service struct {/type Service struct {\n\trepo Repository/' pegasusX/apps/backend-go/driver/service.go
sed -i '' 's/type ServiceConfig struct {/type ServiceConfig struct {\n\tRepo Repository/' pegasusX/apps/backend-go/driver/service.go
sed -i '' 's/now:                c.Now,/repo:                c.Repo,\n\t\tnow:                c.Now,/' pegasusX/apps/backend-go/driver/service.go
