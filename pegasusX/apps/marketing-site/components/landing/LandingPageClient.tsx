"use client";

import { HeroSection } from "@/components/landing/HeroSection";
import { CustomerStrip } from "@/components/landing/CustomerStrip";
import { PlatformThesisBand } from "@/components/landing/PlatformThesisBand";
import { ExperienceSection } from "@/components/landing/ExperienceSection";
import { ControlPlaneSection } from "@/components/landing/ControlPlaneSection";
import { EcosystemSection } from "@/components/landing/EcosystemSection";
import { RolesParadeSection } from "@/components/landing/RolesParadeSection";
import { DispatchEngineSection } from "@/components/landing/DispatchEngineSection";
import { LiveTelemetrySection } from "@/components/landing/LiveTelemetrySection";
import { FinancialIntegritySection } from "@/components/landing/FinancialIntegritySection";
import { ReliabilitySection } from "@/components/landing/ReliabilitySection";
import { TrustProofSection } from "@/components/landing/TrustProofSection";
import { ComponentSystemSection } from "@/components/landing/ComponentSystemSection";
import { CtaSection } from "@/components/landing/CtaSection";
import { Nav } from "@/components/layout/Nav";
import { useScrollSpy } from "@/components/layout/useScrollSpy";

export function LandingPageClient() {
  const activeSection = useScrollSpy();

  return (
    <>
      <Nav activeSection={activeSection} />
      <main>
        <HeroSection />
        <CustomerStrip />
        <PlatformThesisBand />
        <ExperienceSection />
        <ControlPlaneSection />
        <EcosystemSection />
        <RolesParadeSection />
        <DispatchEngineSection />
        <LiveTelemetrySection />
        <FinancialIntegritySection />
        <ReliabilitySection />
        <TrustProofSection />
        <ComponentSystemSection />
        <CtaSection />
      </main>
    </>
  );
}
