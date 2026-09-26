import re

with open("apps/backend-go/driver/cash_bag.go", "r") as f:
    content = f.read()

pattern = re.compile(r'\treconID := uuid\.NewString\(\)')
replacement = r"""	// Deterministic UUID to prevent duplicates on network retries
	reconID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(driverID+":"+shiftDate.String())).String()"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/driver/cash_bag.go", "w") as f:
    f.write(content)
