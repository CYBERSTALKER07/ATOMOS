import SetupWizardShell from "@/components/setup/SetupWizardShell";
import "./setup-onboarding.css";

export default function SetupLayout({ children }: { children: React.ReactNode }) {
  return <SetupWizardShell>{children}</SetupWizardShell>;
}
