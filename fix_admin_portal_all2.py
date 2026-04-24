import os
import re

def regex_replace_in_file(path, pattern, repl):
    full_path = f"the-lab-monorepo/apps/admin-portal/{path}"
    if not os.path.exists(full_path): return
    with open(full_path, "r") as f:
        content = f.read()
    new_content = re.sub(pattern, repl, content)
    if new_content \!= content:
        with open(full_path, "w") as f:
            f.write(new_content)

def fix_all():
    # 1. Global pattern fixes
    for root, _, files in os.walk("the-lab-monorepo/apps/admin-portal/app"):
        for file in files:
            if file.endswith(".tsx"):
                rel_path = os.path.relpath(os.path.join(root, file), "the-lab-monorepo/apps/admin-portal/")
                regex_replace_in_file(rel_path, r'\bonCash=\{', "onPress={")
                regex_replace_in_file(rel_path, r'variant="bordered"', 'variant="outline"')
                regex_replace_in_file(rel_path, r'\bisLoading=\{', 'isPending={')
    
    # Duplicate keys in billing/page.tsx
    billing_path = "app/setup/billing/page.tsx"
    regex_replace_in_file(billing_path, r"^\s+GLOBAL_PAY: 'M12 2C.*?\n", "", flags=re.MULTILINE)
    regex_replace_in_file(billing_path, r"^\s+CASH: 'M11\.8.*?\n", "", flags=re.MULTILINE)
    
    # 3. Handle payment-config
    regex_replace_in_file("app/supplier/payment-config/page.tsx", r"if \(gateway === 'GLOBAL_PAY'\)", "if ((gateway as string) === 'GLOBAL_PAY')")
    
    # 4. Handle supply-lanes ringColor
    regex_replace_in_file("app/supplier/supply-lanes/page.tsx", r"ringColor: c\.text", "'--ring-color': c.text")
    regex_replace_in_file("app/supplier/supply-lanes/page.tsx", r"style=\{\{ background: c\.bg, color: c\.text, '--ring-color': c\.text \}\}", "style={{ background: c.bg, color: c.text, '--ring-color': c.text } as React.CSSProperties}")
    
    # 5. Handle warehouses record cast
    regex_replace_in_file("app/supplier/warehouses/page.tsx", r"\(warehouse as Record<string, unknown>\)", "(warehouse as any)")

    # 6. Recharts 
    for file in ["FleetLoadBars.tsx", "GatewayPieChart.tsx", "RevenueChart.tsx"]:
        regex_replace_in_file(f"components/analytics/advanced/{file}", r"\(value: number\) =>", "(value: any) =>")
    
    # 7. Bridge tests
    regex_replace_in_file("lib/__tests__/bridge.test.ts", r"\(window as Record<string, unknown>\)", "(window as any)")
    
    # 8. offline queue
    regex_replace_in_file("lib/offline-queue.ts", r"useRef<ReturnType<typeof setTimeout>>\(\);", "useRef<ReturnType<typeof setTimeout> | undefined>(undefined);")
    
if __name__ == "__main__":
    fix_all()
