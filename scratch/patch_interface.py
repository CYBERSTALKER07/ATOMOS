import sys

with open(sys.argv[1], 'r') as f:
    lines = f.readlines()

for i, line in enumerate(lines):
    if line.strip() == "ListAudit(ctx context.Context, limit int) ([]AuditRow, error)":
        lines.insert(i + 1, "\tListFeatureFlags(ctx context.Context, tenantType, tenantID string) ([]FeatureFlag, error)\n\tSetFeatureFlag(ctx context.Context, flag FeatureFlag) error\n")
        break

with open(sys.argv[1], 'w') as f:
    f.writelines(lines)
