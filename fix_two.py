with open('the-lab-monorepo/apps/backend-go/analytics/revenue.go', 'r') as f:
    text = f.read()

import re
text = re.sub(r'\s+Cash\s+int64\s+`json:"cash"`', '', text, count=1)
text = text.replace('var total, global_pay, cash, card, cash spanner.NullInt64', 'var total, global_pay, cash, card spanner.NullInt64')
text = text.replace('row.Columns(&day, &total, &global_pay, &cash, &card, &cash)', 'row.Columns(&day, &total, &global_pay, &cash, &card)')

with open('the-lab-monorepo/apps/backend-go/analytics/revenue.go', 'w') as f:
    f.write(text)

print("done")
