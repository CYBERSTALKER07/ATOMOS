import sys

with open(sys.argv[1], 'r') as f:
    lines = f.readlines()

for i, line in enumerate(lines):
    if "pr.Post(\"/tenants/{tenantType}/{tenantID}/transition\", h.HandleTransitionTenant)" in line:
        lines.insert(i + 1, '\t\tpr.Get("/tenants/{tenantType}/{tenantID}/flags", h.HandleListFeatureFlags)\n\t\tpr.Put("/tenants/{tenantType}/{tenantID}/flags/{flagKey}", h.HandleSetFeatureFlag)\n')
        break

with open(sys.argv[1], 'w') as f:
    f.writelines(lines)
