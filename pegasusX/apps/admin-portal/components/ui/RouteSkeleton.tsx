import React from "react";

export function RouteSkeleton({ title = "Loading page..." }: { title?: string }) {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="border-b pb-4">
        <div className="h-6 w-48 rounded bg-gray-200" />
        <div className="mt-2 h-3 w-80 rounded bg-gray-100" />
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-24 rounded-xl border bg-white p-4 space-y-2">
            <div className="h-3 w-20 rounded bg-gray-200" />
            <div className="h-6 w-16 rounded bg-gray-300" />
          </div>
        ))}
      </div>

      <div className="h-64 rounded-xl border bg-white p-4 space-y-3">
        <div className="h-4 w-32 rounded bg-gray-200" />
        <div className="space-y-2 pt-2">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="h-8 rounded bg-gray-100" />
          ))}
        </div>
      </div>
      <p className="sr-only">{title}</p>
    </div>
  );
}
