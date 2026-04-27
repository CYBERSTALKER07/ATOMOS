import sys

with open("the-lab-monorepo/apps/backend-go/main.go", "r") as f:
    content = f.read()

content = content.replace(
    "retailerPricingSvc := supplier.NewRetailerPricingService(spannerClient, app.SpannerRouter, svc.Producer)",
    "retailerPricingSvc := supplier.NewRetailerPricingService(spannerClient, app.SpannerRouter, svc.Producer, app.Cache)"
)

content = content.replace(
    "auth.HandleCreateFamilyMember(spannerClient)(w, r)",
    "func(w http.ResponseWriter, r *http.Request) { invalidate := func(ctx context.Context, keys ...string) { if app.Cache \!= nil { app.Cache.Invalidate(ctx, keys...) } }; auth.HandleCreateFamilyMember(spannerClient, invalidate)(w, r) }(w, r)"
)

content = content.replace(
    "auth.HandleDeleteFamilyMember(spannerClient)",
    "auth.HandleDeleteFamilyMember(spannerClient, func(ctx context.Context, keys ...string) { if app.Cache \!= nil { app.Cache.Invalidate(ctx, keys...) } })"
)

content = content.replace(
    "empathySvc := &settings.EmpathyService{Client: spannerClient}",
    "empathySvc := &settings.EmpathyService{Client: spannerClient, Cache: app.Cache}"
)

with open("the-lab-monorepo/apps/backend-go/main.go", "w") as f:
    f.write(content)
