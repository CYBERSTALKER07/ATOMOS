import os
import re

def replace_in_file(path, old, new):
    full_path = f"the-lab-monorepo/apps/admin-portal/{path}"
    if not os.path.exists(full_path): return
    with open(full_path, "r") as f:
        content = f.read()
    new_content = content.replace(old, new)
    if new_content \!= content:
        with open(full_path, "w") as f:
            f.write(new_content)

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
    for root, _, files in os.walk("the-lab-monorepo/apps/admin-portal/app"):
        for file in files:
            if file.endswith(".tsx"):
                rel_path = os.path.relpath(os.path.join(root, file), "the-lab-monorepo/apps/admin-portal/")
                replace_in_file(rel_path, "onCash={", "onPress={")
                replace_in_file(rel_path, "onCash(", "onPress(")
                replace_in_file(rel_path, 'variant="bordered"', 'variant="outline"')
                replace_in_file(rel_path, 'isLoading={', 'isPending={')

    replace_in_file("components/analytics/advanced/FleetLoadBars.tsx", "formatter={(value: number) =>", "formatter={(value: any) =>")
    replace_in_file("components/analytics/advanced/GatewayPieChart.tsx", "formatter={(value: number) =>", "formatter={(value: any) =>")
    replace_in_file("components/analytics/advanced/RevenueChart.tsx", "formatter={(value: number) =>", "formatter={(value: any) =>")
    replace_in_file("lib/__tests__/bridge.test.ts", "window as Record<string, unknown>", "window as any")
    replace_in_file("app/supplier/supply-lanes/page.tsx", "(lane.sourceNode as any).lat", "(lane.sourceNode as any)?.lat")
    replace_in_file("app/supplier/supply-lanes/page.tsx", "(lane.targetNode as any).lng", "(lane.targetNode as any)?.lng")
    replace_in_file("app/supplier/supply-lanes/page.tsx", "(lane.sourceNode as any).lng", "(lane.sourceNode as any)?.lng")
    replace_in_file("app/supplier/supply-lanes/page.tsx", "(lane.targetNode as any).lat", "(lane.targetNode as any)?.lat")

if __name__ == "__main__":
    fix_all()
