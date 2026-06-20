import SetupWizardShell from "@/components/setup/SetupWizardShell";

export default function SetupLayout({ children }: { children: React.ReactNode }) {
  return <SetupWizardShell>{children}</SetupWizardShell>;
}
