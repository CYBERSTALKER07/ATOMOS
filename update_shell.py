import re

path = '/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/supplier-portal/components/SupplierShell.tsx'
with open(path, 'r') as f:
    content = f.read()

# Replace NAV array
nav_replacement = """const NAV: NavSection[] = [
  {
    items: [{ href: "/dashboard", icon: "overview", label: "Overview" }],
  },
  {
    label: "Operations",
    items: [
      { href: "/orders", icon: "orders", label: "Orders" },
      { href: "/dispatch", icon: "dispatch", label: "Dispatch" },
      { href: "/manifests", icon: "manifests", label: "Manifests" },
      { href: "/fleet", icon: "fleet", label: "Fleet" },
      { href: "/exceptions", icon: "warning", label: "Exceptions" },
    ],
  },
  {
    label: "Catalog",
    items: [
      { href: "/inventory", icon: "inventory", label: "Inventory" },
      { href: "/catalog", icon: "catalog", label: "Catalog" },
      { href: "/pricing", icon: "pricing", label: "Pricing" },
    ],
  },
  {
    label: "Network",
    items: [
      { href: "/topology", icon: "topology", label: "Topology" },
      { href: "/factories", icon: "factory", label: "Factories" },
      { href: "/warehouses", icon: "warehouse", label: "Warehouses" },
      { href: "/delivery-zones", icon: "global", label: "Delivery Zones" },
      { href: "/supply-lanes", icon: "fleet", label: "Supply Lanes" },
    ],
  },
  {
    label: "Finance",
    items: [
      { href: "/treasury", icon: "treasury", label: "Treasury" },
      { href: "/reconciliation", icon: "reconcile", label: "Reconciliation" },
      { href: "/payments", icon: "payment", label: "Payments" },
      { href: "/earnings", icon: "pricing", label: "Earnings" },
    ],
  },
  {
    label: "Settings",
    items: [
      { href: "/profile", icon: "supplier", label: "Profile" },
      { href: "/org-fleet", icon: "person-add", label: "Org & Fleet" },
      { href: "/returns", icon: "returns", label: "Returns" },
    ],
  },
];"""

content = re.sub(r'const NAV: NavSection\[\].*?\];', nav_replacement, content, flags=re.DOTALL)

# Update isActiveRoute
content = re.sub(
    r'function isActiveRoute\(pathname: string, href: string\): boolean \{.*?\}',
    """function isActiveRoute(pathname: string, href: string): boolean {
  if (href === "/dashboard") {
    return pathname === "/dashboard" || pathname === "/";
  }
  return pathname === href || pathname.startsWith(`${href}/`);
}""",
    content,
    flags=re.DOTALL
)

# Update buildBreadcrumbs
content = re.sub(
    r'function buildBreadcrumbs\(pathname: string\): \{ label: string; href: string \}\[\] \{.*?\}',
    """function buildBreadcrumbs(pathname: string): { label: string; href: string }[] {
  if (pathname === "/dashboard" || pathname === "/") {
    return [{ label: "Overview", href: "/dashboard" }];
  }
  const crumbs: { label: string; href: string }[] = [
    { label: "Home", href: "/dashboard" },
  ];
  let path = "";
  for (const seg of pathname.split("/").filter(Boolean)) {
    path += `/${seg}`;
    crumbs.push({
      label: seg.charAt(0).toUpperCase() + seg.slice(1).replace(/-/g, " "),
      href: path,
    });
  }
  return crumbs;
}""",
    content,
    flags=re.DOTALL
)

# Update BARE_ROUTES
content = re.sub(
    r"const BARE_ROUTES = \['/login', '/signup', '/auth/'\];",
    'const BARE_ROUTES = ["/auth/", "/setup/billing"];',
    content
)

# Replace useAuth with readTokenFromCookie
content = content.replace("import { useAuth } from '@/hooks/useAuth';", "import { clearSession, readTokenFromCookie } from '@/lib/auth';")
content = content.replace("export default function AdminShell({ children }: { children: React.ReactNode }) {", "export default function SupplierShell({ children }: { children: React.ReactNode }) {")

# Remove isAuthRef and useAuth hook calls
content = re.sub(r'const isAuthRef = useRef\(true\);\s*const \{ isGlobalAdmin, isFactoryStaff, supplierRole \} = useAuth\(\);', '', content)
content = re.sub(r'const cookies = document.cookie;\s*isAuthRef\.current = cookies\.includes\([^;]+;\s*\}', '', content, flags=re.DOTALL)
content = re.sub(r'useEffect\(\(\) => \{\s*const cookies.*?\}, \[pathname\]\);', '', content, flags=re.DOTALL)

# Handle filteredNav
content = re.sub(
    r'const filteredNav = useMemo\(\(\) =>[\s\S]*?\[isGlobalAdmin, isFactoryStaff\],\s*\);',
    'const filteredNav = NAV;',
    content
)

# Handle DrawerContent signature
content = re.sub(
    r'isGlobalAdmin,\s*isFactoryStaff,\s*',
    '',
    content
)
content = re.sub(
    r'isGlobalAdmin: boolean;\s*isFactoryStaff: boolean;\s*',
    '',
    content
)

# Handle handleLogout
content = re.sub(
    r'const handleLogout = useCallback\(\(\) => \{[\s\S]*?window\.location\.href = \'/auth/login\';\s*\}, \[\]\);',
    """const handleLogout = useCallback(() => {
    clearSession();
    window.location.href = '/auth/login';
  }, []);""",
    content
)

# Replace "Pegasus Hub" -> "pegasusX Supplier Hub"
content = content.replace('Pegasus Hub', 'Supplier Hub')
content = content.replace('Enterprise', 'pegasusX')

# Replace role display
content = re.sub(
    r'\{supplierRole === \'NODE_ADMIN\'.*?\}',
    '{typeof document !== "undefined" && Boolean(readTokenFromCookie()) ? "Administrator" : "Guest"}',
    content
)
content = content.replace('<span className="desk-profile-name">Admin</span>', '<span className="desk-profile-name">Supplier</span>')
content = content.replace('<div className="desk-profile-avatar">AS</div>', '<div className="desk-profile-avatar">SP</div>')

with open(path, 'w') as f:
    f.write(content)
