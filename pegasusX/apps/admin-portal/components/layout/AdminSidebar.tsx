"use client";

import React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";

interface NavItem {
  href: string;
  label: string;
  badge?: string | number;
}

interface NavSection {
  title: string;
  items: NavItem[];
}

export function AdminSidebar({
  isOpen,
  onClose,
  pendingFlagsCount,
  dlqCount,
}: {
  isOpen: boolean;
  onClose: () => void;
  pendingFlagsCount?: number;
  dlqCount?: number;
}) {
  const pathname = usePathname();

  const sections: NavSection[] = [
    {
      title: "Overview & Operations",
      items: [
        {
          href: "/overview",
          label: "Command Cockpit",
          badge: dlqCount && dlqCount > 0 ? `${dlqCount} DLQ` : undefined,
        },
        { href: "/infra", label: "Cloud & Cell Infra" },
        { href: "/finops", label: "FinOps & Billing" },
        { href: "/telemetry", label: "Operational Logs" },
        { href: "/analytics", label: "Ecosystem Scorecards" },
      ],
    },
    {
      title: "Supply Chain Traceability",
      items: [
        { href: "/traceability/recall", label: "Lot Recall Funnel" },
        { href: "/traceability/sandbox", label: "Scenario Sandboxes" },
        { href: "/traceability/actions", label: "Kinetic Action Guards" },
      ],
    },
    {
      title: "Logistics & Fleet Safety",
      items: [
        { href: "/logistics/dvir", label: "DVIR Red-Tag Station" },
        { href: "/logistics/gate", label: "Yard Gate Clearance" },
      ],
    },
    {
      title: "Treasury & Compliance",
      items: [
        { href: "/compliance/cashier", label: "Cashier Blind Vault" },
        { href: "/compliance/aml", label: "AML Cash Check" },
        { href: "/compliance/fiscal", label: "Soliq Fiscal Check" },
      ],
    },
    {
      title: "Platform Governance",
      items: [
        {
          href: "/governance/flags",
          label: "Dual-Control Flags",
          badge: pendingFlagsCount && pendingFlagsCount > 0 ? `${pendingFlagsCount} Pending` : undefined,
        },
        { href: "/governance/audit", label: "Audit Ledger" },
        { href: "/governance/tenants", label: "Tenants & Catalog" },
      ],
    },
  ];

  return (
    <>
      {/* Mobile overlay */}
      {isOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={onClose}
          aria-hidden="true"
        />
      )}

      <aside
        className={`fixed inset-y-0 left-0 z-50 flex w-64 flex-col border-r bg-white transition-transform duration-200 ease-in-out lg:static lg:translate-x-0 ${
          isOpen ? "translate-x-0" : "-translate-x-full"
        }`}
      >
        <div className="flex h-16 items-center justify-between border-b px-5">
          <Link href="/overview" className="flex items-center gap-2">
            <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-600 font-bold text-white text-base">
              P
            </span>
            <div>
              <span className="text-sm font-bold tracking-tight text-gray-900 block leading-none">
                PegasusX
              </span>
              <span className="text-[10px] font-semibold tracking-wider uppercase text-indigo-600 block mt-0.5">
                Owner Tower
              </span>
            </div>
          </Link>
          <button
            type="button"
            onClick={onClose}
            className="rounded p-1 text-gray-400 hover:text-gray-700 lg:hidden"
          >
            ✕
          </button>
        </div>

        <nav className="flex-1 overflow-y-auto px-3 py-4 space-y-6">
          {sections.map((sec) => (
            <div key={sec.title}>
              <h3 className="px-3 text-[10px] font-bold uppercase tracking-wider text-gray-600 mb-2">
                {sec.title}
              </h3>
              <ul className="space-y-1">
                {sec.items.map((item) => {
                  const isActive = pathname === item.href;
                  return (
                    <li key={item.href}>
                      <Link
                        href={item.href}
                        onClick={onClose}
                        className={`group flex items-center justify-between rounded-lg px-3 py-2 text-xs font-medium transition-colors ${
                          isActive
                            ? "bg-indigo-50 text-indigo-700 font-semibold"
                            : "text-gray-700 hover:bg-gray-100 hover:text-gray-900"
                        }`}
                      >
                        <span>{item.label}</span>
                        {item.badge && (
                          <span
                            className={`rounded-full px-2 py-0.5 text-[10px] font-bold ${
                              String(item.badge).includes("DLQ")
                                ? "bg-red-100 text-red-700"
                                : "bg-amber-100 text-amber-700"
                            }`}
                          >
                            {item.badge}
                          </span>
                        )}
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
        </nav>

        <div className="border-t p-3 bg-gray-50/50">
          <div className="flex items-center justify-between px-2 text-[11px] text-gray-500">
            <span>Cell: cell-uz</span>
            <span className="font-mono">v1.2.0-prod</span>
          </div>
        </div>
      </aside>
    </>
  );
}
