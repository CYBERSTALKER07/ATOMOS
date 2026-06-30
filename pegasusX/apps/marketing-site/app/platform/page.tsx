import Link from "next/link";
import { Nav } from "@/components/layout/Nav";
import { ROLES, CAPABILITIES, SOLUTIONS } from "@/lib/constants";

const HOW_IT_WORKS = [
  {
    title: "Apps for every team",
    items: [
      "Supplier dashboards for network oversight",
      "Warehouse dispatch boards and fleet maps",
      "Factory loading and manifest tracking",
      "Driver routes with stop-by-stop guidance",
      "Retailer ordering and live tracking",
      "Gate seal control before departure",
    ],
  },
  {
    title: "One live picture",
    items: [
      "Order status every team agrees on",
      "Dispatch boards that update as work happens",
      "Fleet maps with honest on-time vs. delayed status",
      "Payments tracked from checkout to settlement",
    ],
  },
  {
    title: "Built for busy operations",
    items: [
      "Handles peak dispatch windows without slowdown",
      "Updates reach every app within seconds",
      "Safe during high-volume order days",
      "Works on web, desktop, and mobile",
    ],
  },
  {
    title: "Accountability at every handoff",
    items: [
      "Orders vetted before fulfillment",
      "Loads sealed before trucks leave",
      "Driver progress tracked against plan",
      "Delivery proof attached to every stop",
    ],
  },
];

export default function PlatformPage() {
  return (
    <>
      <Nav />
      <main className="mx-auto max-w-7xl px-4 py-16 md:px-6">
        <p className="mkt-section-label">Platform</p>
        <h1 className="mkt-display mt-3 max-w-4xl text-4xl md:text-6xl">
          How Pegasus connects your network
        </h1>
        <p className="mt-6 max-w-3xl text-lg text-[var(--mkt-muted)]">
          Six dedicated apps — one shared platform. From the moment a retailer places an order
          to the moment a driver confirms delivery, every team sees the same live picture.
        </p>

        <div className="mt-16 grid gap-6 md:grid-cols-2">
          {HOW_IT_WORKS.map((layer) => (
            <div key={layer.title} className="mkt-card p-6">
              <h2 className="text-lg font-semibold">{layer.title}</h2>
              <ul className="mt-4 space-y-2 text-sm text-[var(--mkt-muted)]">
                {layer.items.map((item) => (
                  <li key={item} className="flex gap-2">
                    <span className="text-[var(--mkt-subtle)]">→</span>
                    {item}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <section className="mt-20">
          <h2 className="text-2xl font-semibold">A typical day in your network</h2>
          <ol className="mt-6 space-y-4">
            {[
              "A retailer places an order through their catalog",
              "The supplier reviews and approves it",
              "The warehouse dispatches it to a truck on the morning board",
              "The gate team seals the load before departure",
              "The driver follows the route stop by stop",
              "The retailer tracks progress live — no phone call needed",
              "Payment is confirmed and treasury reflects the collection",
            ].map((step, i) => (
              <li key={step} className="flex gap-4">
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-[var(--mkt-border-strong)] text-sm font-semibold">
                  {i + 1}
                </span>
                <p className="pt-1 text-[var(--mkt-muted)]">{step}</p>
              </li>
            ))}
          </ol>
        </section>

        <section className="mt-20">
          <div className="flex items-end justify-between gap-4">
            <h2 className="text-2xl font-semibold">Solutions</h2>
            <Link href="/solutions" className="text-sm text-[var(--mkt-muted)] hover:text-[var(--mkt-text)]">
              View all →
            </Link>
          </div>
          <div className="mt-6 grid gap-4 sm:grid-cols-2">
            {SOLUTIONS.map((solution) => (
              <Link
                key={solution.slug}
                href={`/solutions/${solution.slug}`}
                className="mkt-card block p-5 transition hover:border-[var(--mkt-border-strong)]"
              >
                <h3 className="font-semibold">{solution.title}</h3>
                <p className="mt-2 text-sm text-[var(--mkt-muted)]">{solution.summary}</p>
              </Link>
            ))}
          </div>
        </section>

        <section className="mt-16">
          <div className="flex items-end justify-between gap-4">
            <h2 className="text-2xl font-semibold">Capabilities</h2>
            <Link href="/capabilities" className="text-sm text-[var(--mkt-muted)] hover:text-[var(--mkt-text)]">
              View all →
            </Link>
          </div>
          <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {CAPABILITIES.map((cap) => (
              <Link
                key={cap.slug}
                href={`/capabilities/${cap.slug}`}
                className="mkt-card block p-5 transition hover:border-[var(--mkt-border-strong)]"
              >
                <h3 className="font-semibold">{cap.title}</h3>
              </Link>
            ))}
          </div>
        </section>

        <section className="mt-16">
          <h2 className="text-2xl font-semibold">Who it&apos;s for</h2>
          <div className="mt-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {ROLES.map((role) => (
              <Link
                key={role.slug}
                href={`/roles/${role.slug}`}
                className="mkt-card flex items-center gap-3 p-4 transition hover:border-[var(--mkt-border-strong)]"
              >
                <span className="role-badge__dot" />
                <div>
                  <p className="font-medium">{role.name}</p>
                  <p className="text-xs text-[var(--mkt-subtle)]">{role.tagline}</p>
                </div>
              </Link>
            ))}
          </div>
        </section>

        <div className="mt-16">
          <Link href="/contact" className="mkt-btn mkt-btn-primary">
            Request a demo
          </Link>
        </div>
      </main>
    </>
  );
}
