import Link from "next/link";
import { Nav } from "@/components/layout/Nav";
import { PageIntro } from "@/components/layout/PageIntro";
import { ROLES } from "@/lib/constants";

export default function RolesIndexPage() {
  return (
    <>
      <Nav />
      <main className="mx-auto max-w-7xl px-4 py-16 md:px-6">
        <PageIntro
          label="Who it's for"
          title="Six teams. One connected network."
          description="Every person in your supply chain gets tools built for their job — all reading from the same live picture."
        />
        <div className="mt-16 grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {ROLES.map((role) => (
            <Link
              key={role.slug}
              href={`/roles/${role.slug}`}
              className="mkt-card block p-6 transition hover:border-[var(--mkt-border-strong)]"
            >
              <div className="flex items-center gap-3">
                <span className="role-badge__dot" />
                <h2 className="text-xl font-semibold">{role.name}</h2>
              </div>
              <p className="mt-3 text-sm text-[var(--mkt-muted)]">{role.tagline}</p>
              <p className="mt-4 text-xs text-[var(--mkt-subtle)]">
                {role.surfaces.join(" · ")}
              </p>
            </Link>
          ))}
        </div>
      </main>
    </>
  );
}
