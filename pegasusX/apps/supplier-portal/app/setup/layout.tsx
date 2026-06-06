export default function SetupLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="setup-shell">
      <div className="setup-inner">{children}</div>
    </div>
  );
}
