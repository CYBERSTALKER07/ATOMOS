import type { SVGProps } from "react";

type IconProps = SVGProps<SVGSVGElement> & { size?: number };

function IconBase({ size = 28, children, ...props }: IconProps & { children: React.ReactNode }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
      {...props}
    >
      {children}
    </svg>
  );
}

export function IconGo(props: IconProps) {
  return (
    <IconBase {...props}>
      <circle cx="12" cy="12" r="9" />
      <path d="M8 12h8M12 8v8" />
    </IconBase>
  );
}

export function IconSpanner(props: IconProps) {
  return (
    <IconBase {...props}>
      <ellipse cx="12" cy="12" rx="9" ry="6" />
      <path d="M6 12h12" />
    </IconBase>
  );
}

export function IconRedis(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="M4 8l8-4 8 4-8 4-8-4z" />
      <path d="M4 12l8 4 8-4" />
      <path d="M4 16l8 4 8-4" />
    </IconBase>
  );
}

export function IconKafka(props: IconProps) {
  return (
    <IconBase {...props}>
      <circle cx="12" cy="6" r="2" fill="currentColor" stroke="none" />
      <circle cx="6" cy="14" r="2" fill="currentColor" stroke="none" />
      <circle cx="18" cy="14" r="2" fill="currentColor" stroke="none" />
      <circle cx="12" cy="18" r="2" fill="currentColor" stroke="none" />
      <path d="M12 8v2M10 13l-2 1M14 13l2 1M12 16v-2" />
    </IconBase>
  );
}

export function IconKubernetes(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="M12 3l7 4v6l-7 4-7-4V7l7-4z" />
      <circle cx="12" cy="12" r="2" fill="currentColor" stroke="none" />
    </IconBase>
  );
}

export function IconNextjs(props: IconProps) {
  return (
    <IconBase {...props}>
      <circle cx="12" cy="12" r="9" />
      <path d="M8 16V8l8 8V8" />
    </IconBase>
  );
}

export function IconKotlin(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="M4 18L14 6h6L10 18H4z" />
    </IconBase>
  );
}

export function IconSwift(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="M4 18c4-8 12-12 16-14-2 6-6 12-16 14z" />
    </IconBase>
  );
}

export function IconWebsocket(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="M4 12a8 8 0 0116 0" />
      <path d="M7 12a5 5 0 0110 0" />
      <circle cx="12" cy="12" r="1.5" fill="currentColor" stroke="none" />
    </IconBase>
  );
}

export function IconH3(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="M12 4l6 3.5v7L12 18l-6-3.5v-7L12 4z" />
    </IconBase>
  );
}

export function IconMap(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="M4 6l6-2 6 2 4-2v14l-4 2-6-2-6 2-4-2V6z" />
      <path d="M10 4v14M16 6v14" />
    </IconBase>
  );
}

export function IconNetwork(props: IconProps) {
  return (
    <IconBase {...props}>
      <circle cx="6" cy="6" r="2" fill="currentColor" stroke="none" />
      <circle cx="18" cy="6" r="2" fill="currentColor" stroke="none" />
      <circle cx="12" cy="18" r="2" fill="currentColor" stroke="none" />
      <path d="M7.5 7.5L10.5 16M16.5 7.5L13.5 16M8 6h8" />
    </IconBase>
  );
}

export type TechIconId =
  | "go"
  | "spanner"
  | "redis"
  | "kafka"
  | "kubernetes"
  | "nextjs"
  | "kotlin"
  | "swift"
  | "websocket"
  | "h3"
  | "map"
  | "network";

export const TECH_ICONS: Record<
  TechIconId,
  { label: string; Icon: (props: IconProps) => React.JSX.Element }
> = {
  go: { label: "Go", Icon: IconGo },
  spanner: { label: "Spanner", Icon: IconSpanner },
  redis: { label: "Redis", Icon: IconRedis },
  kafka: { label: "Kafka", Icon: IconKafka },
  kubernetes: { label: "Maglev", Icon: IconKubernetes },
  nextjs: { label: "Next.js", Icon: IconNextjs },
  kotlin: { label: "Kotlin", Icon: IconKotlin },
  swift: { label: "Swift", Icon: IconSwift },
  websocket: { label: "WebSocket", Icon: IconWebsocket },
  h3: { label: "H3", Icon: IconH3 },
  map: { label: "Map", Icon: IconMap },
  network: { label: "Network", Icon: IconNetwork },
};
