import { AlertTriangle } from "lucide-react";
import { normalizeReceivingWindow } from "../../lib/receiving-window";

export function ProfileField({
  label,
  value,
  icon: Icon,
  editing,
  errorMessage,
  onChange,
}: {
  label: string;
  value: string;
  icon: React.ElementType;
  editing: boolean;
  errorMessage?: string;
  onChange: (v: string) => void;
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-2 text-[var(--desk-text-tertiary)]">
        <Icon size={14} />
        <span className="md-typescale-label-small font-light uppercase tracking-widest">
          {label}
        </span>
      </div>
      {editing ? (
        <>
          <input
            value={value}
            onChange={(e) => onChange(e.target.value)}
            aria-invalid={Boolean(errorMessage)}
            className={`w-full h-11 px-4 bg-[var(--desk-canvas)] rounded-xl outline-none transition-all md-typescale-body-medium font-light text-[var(--desk-text-primary)] ${
              errorMessage
                ? "ring-2 ring-red-200 border border-red-300"
                : "focus:ring-2 focus:ring-[var(--desk-accent-soft)]"
            }`}
          />
          {errorMessage && (
            <p className="mt-1 flex items-center gap-1 text-[11px] font-light text-red-600 uppercase tracking-wide">
              <AlertTriangle size={10} />
              {errorMessage}
            </p>
          )}
        </>
      ) : (
        <p className="md-typescale-body-large font-light text-[var(--desk-text-primary)] pl-0.5">
          {value || "UNSET"}
        </p>
      )}
    </div>
  );
}

export function ProfileTimeField({
  label,
  value,
  icon: Icon,
  editing,
  errorMessage,
  onChange,
}: {
  label: string;
  value: string;
  icon: React.ElementType;
  editing: boolean;
  errorMessage?: string;
  onChange: (v: string) => void;
}) {
  const displayValue = normalizeReceivingWindow(value);

  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-2 text-[var(--desk-text-tertiary)]">
        <Icon size={14} />
        <span className="md-typescale-label-small font-light uppercase tracking-widest">
          {label}
        </span>
      </div>
      {editing ? (
        <>
          <input
            type="time"
            value={displayValue}
            onChange={(e) => onChange(e.target.value)}
            aria-invalid={Boolean(errorMessage)}
            className={`w-full h-11 px-4 bg-[var(--desk-canvas)] rounded-xl outline-none transition-all md-typescale-body-medium font-light text-[var(--desk-text-primary)] ${
              errorMessage
                ? "ring-2 ring-red-200 border border-red-300"
                : "focus:ring-2 focus:ring-[var(--desk-accent-soft)]"
            }`}
          />
          {errorMessage && (
            <p className="mt-1 flex items-center gap-1 text-[11px] font-light text-red-600 uppercase tracking-wide">
              <AlertTriangle size={10} />
              {errorMessage}
            </p>
          )}
        </>
      ) : (
        <p className="md-typescale-body-large font-light text-[var(--desk-text-primary)] pl-0.5">
          {displayValue || "UNSET"}
        </p>
      )}
    </div>
  );
}
