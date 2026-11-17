"use client";

import { Suspense, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";

function safeReturnToPath(raw: string | null): string {
  if (!raw) return "/overview";
  try {
    const decoded = decodeURIComponent(raw).trim();
    if (!decoded.startsWith("/") || decoded.startsWith("//") || decoded.startsWith("/\\")) {
      return "/overview";
    }
    const check = new URL(decoded, "https://pegasusx.local");
    if (check.origin !== "https://pegasusx.local") {
      return "/overview";
    }
    return `${check.pathname}${check.search}${check.hash}` || "/overview";
  } catch {
    return "/overview";
  }
}

function RootRedirector() {
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    const returnTo = safeReturnToPath(searchParams.get("return_to"));
    router.replace(returnTo);
  }, [router, searchParams]);

  return (
    <div className="flex h-64 items-center justify-center">
      <div className="text-center space-y-2">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent mx-auto" />
        <p className="text-xs text-gray-500">Connecting to PegasusX Control Tower...</p>
      </div>
    </div>
  );
}

export default function HomePage() {
  return (
    <Suspense
      fallback={
        <div className="flex h-64 items-center justify-center">
          <div className="text-center space-y-2">
            <div className="h-6 w-6 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent mx-auto" />
            <p className="text-xs text-gray-500">Connecting to PegasusX Control Tower...</p>
          </div>
        </div>
      }
    >
      <RootRedirector />
    </Suspense>
  );
}
