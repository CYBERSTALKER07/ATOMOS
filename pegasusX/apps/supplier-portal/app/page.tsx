export default function HomePage() {
  return (
    <main className="min-h-screen p-12">
      <h1 className="md-typescale-display-large">pegasusX</h1>
      <p className="md-typescale-body-large mt-4" style={{ color: "var(--color-md-outline)" }}>
        Supplier control plane. Single tenant. Thousands of retailers.
      </p>
      <div className="mt-8 flex gap-4">
        <a className="md-btn md-btn-filled" href="/auth/register">Set up supplier</a>
        <a className="md-btn md-btn-outlined" href="/setup/billing">Billing</a>
        <a className="md-btn md-btn-outlined" href="/org-fleet">Org &amp; Fleet</a>
        <a className="md-btn md-btn-outlined" href="/payments">Payments</a>
        <a className="md-btn md-btn-outlined" href="/ai/recommendations">AI Review</a>
      </div>
    </main>
  );
}
