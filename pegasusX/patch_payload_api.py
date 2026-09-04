import re

with open("apps/payload-terminal/api.ts", "r") as f:
    content = f.read()

pattern = re.compile(r'export async function reportScanProgress\(manifestId: string, itemId: string, itemVu: number\): Promise<\{ loaded_vu: number \}> \{\n\s*const res = await authFetch\(`\$\{API_BASE\}/v1/payload/scan`, \{\n\s*method: \'POST\',\n\s*headers: \{ \'Content-Type\': \'application/json\' \},\n\s*body: JSON\.stringify\(\{ manifest_id: manifestId, item_id: itemId, item_vu: itemVu \}\),\n\s*\}\);')
replacement = r"""export async function reportScanProgress(manifestId: string, itemId: string, itemVu: number): Promise<{ loaded_vu: number }> {
    const res = await authFetch(`${API_BASE}/v1/payloader/manifests/${manifestId}/load-ledger/scan`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ order_id: itemId, quantity: itemVu }),
    });"""

content = pattern.sub(replacement, content)

with open("apps/payload-terminal/api.ts", "w") as f:
    f.write(content)
