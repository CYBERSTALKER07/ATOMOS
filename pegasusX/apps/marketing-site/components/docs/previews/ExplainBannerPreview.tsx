"use client";

function MarketingExplainBanner({
  title,
  summary,
  nextSteps,
}: {
  title: string;
  summary: string;
  nextSteps: string[];
}) {
  return (
    <div className="explain-status-banner">
      <p className="explain-status-banner__title">{title}</p>
      <p className="explain-status-banner__summary">{summary}</p>
      <ol className="explain-status-banner__steps">
        {nextSteps.map((step) => (
          <li key={step}>{step}</li>
        ))}
      </ol>
    </div>
  );
}

export function ExplainBannerPreview() {
  return (
    <MarketingExplainBanner
      title="Dispatch frozen"
      summary="Manual intervention window is active. Auto-assignment paused."
      nextSteps={["Review freeze-lock scope", "Release lock when ops complete"]}
    />
  );
}
