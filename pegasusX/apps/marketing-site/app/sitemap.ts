import type { MetadataRoute } from "next";
import { CAPABILITIES, ROLES, SOLUTIONS } from "@/lib/constants";

const BASE_URL = process.env.NEXT_PUBLIC_SITE_URL ?? "https://pegasus.io";

export default function sitemap(): MetadataRoute.Sitemap {
  const staticRoutes = [
    "",
    "/platform",
    "/solutions",
    "/capabilities",
    "/roles",
    "/customers",
    "/about",
    "/contact",
    "/components",
    "/playground",
  ];

  const solutionRoutes = SOLUTIONS.map((s) => `/solutions/${s.slug}`);
  const capabilityRoutes = CAPABILITIES.map((c) => `/capabilities/${c.slug}`);
  const roleRoutes = ROLES.map((r) => `/roles/${r.slug}`);

  const componentRoutes = [
    "motion-tokens",
    "portal-button",
    "pulse-timeline",
    "portal-card",
    "page-chrome",
    "explain-banner",
    "kpi-stat-card",
    "fleet-route-map",
    "topology-graph",
    "status-chip",
    "scroll-section",
    "role-badge",
  ].map((slug) => `/components/${slug}`);

  const allRoutes = [
    ...staticRoutes,
    ...solutionRoutes,
    ...capabilityRoutes,
    ...roleRoutes,
    ...componentRoutes,
  ];

  return allRoutes.map((path) => ({
    url: `${BASE_URL}${path}`,
    lastModified: new Date(),
    changeFrequency: path === "" ? "weekly" : "monthly",
    priority: path === "" ? 1 : 0.7,
  }));
}
