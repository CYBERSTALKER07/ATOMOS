import sys

with open("pegasus/apps/backend-go/adminroutes/routes.go", "r") as f:
    text = f.read()

old_res = """        // 7. P0 — resolve shop-closed escalation (WAIT | BYPASS | RETURN_TO_DEPOT).
        r.HandleFunc("/v1/admin/shop-closed/resolve",
                auth.RequireRole(adminOrSupplier, log(d.Order.HandleResolveShopClosed(d.ShopClosedDeps))))"""

new_res = """        // 7. P0 — resolve shop-closed escalation (WAIT | BYPASS | RETURN_TO_DEPOT).
        r.HandleFunc("/v1/admin/shop-closed/active",
                auth.RequireRole(adminOrSupplier, log(d.Order.HandleListActiveShopClosedAttempts(d.ShopClosedDeps))))
        r.HandleFunc("/v1/admin/shop-closed/resolve",
                auth.RequireRole(adminOrSupplier, log(d.Order.HandleResolveShopClosed(d.ShopClosedDeps))))"""

print(text.replace(old_res, new_res), file=open("pegasus/apps/backend-go/adminroutes/routes.go", "w"))
