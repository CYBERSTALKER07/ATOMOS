with open("pegasus/apps/admin-portal/lib/offline-queue.ts", "r") as f:
    content = f.read()

import re

old_func = r"""export function enqueueMutation\(
  url: string,
  method: string,
  body\?: string,
  headers: Record<string, string> = \{\}
\) \{
  OfflineManager\.enqueue\(\{"""

new_func = """export function enqueueMutation(
  url: string,
  method: string,
  body?: string,
  headers: Record<string, string> = {}
) {
  if (\!headers['Idempotency-Key']) {
    headers['Idempotency-Key'] = crypto.randomUUID();
  }
  
  OfflineManager.enqueue({"""

content = re.sub(old_func, new_func, content)

with open("pegasus/apps/admin-portal/lib/offline-queue.ts", "w") as f:
    f.write(content)

