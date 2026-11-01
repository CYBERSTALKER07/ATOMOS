"use client";

import { Suspense, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";

function RootRedirector() {
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    const returnTo = searchParams.get("return_to");
    if (returnTo) {
      router.replace(decodeURIComponent(returnTo));
    } else {
      router.replace("/overview");
    }
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
