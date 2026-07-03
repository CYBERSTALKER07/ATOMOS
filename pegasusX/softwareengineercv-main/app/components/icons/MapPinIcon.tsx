type MapPinIconProps = {
  className?: string;
  size?: number;
  title?: string;
};

/** Streamline pixel map pin — uses currentColor for themeable fills. */
export default function MapPinIcon({ className = '', size = 20, title }: MapPinIconProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 32 32"
      width={size}
      height={size}
      className={className}
      aria-hidden={title ? undefined : true}
      role={title ? 'img' : undefined}
    >
      {title ? <title>{title}</title> : null}
      <g fill="currentColor">
        <path d="m25.91 14.47 -3.05 0 0 1.53 -1.52 0 0 3.05 1.52 0 0 1.52 3.05 0 0 -1.52 1.52 0 0 -3.05 -1.52 0 0 -1.53z" />
        <path d="M28.95 14.47h1.53V16h-1.53Z" />
        <path d="M27.43 12.95h1.52v1.52h-1.52Z" />
        <path d="M21.34 11.43h6.09v1.52h-6.09Z" />
        <path d="M19.81 12.95h1.53v1.52h-1.53Z" />
        <path d="M18.29 14.47h1.52V16h-1.52Z" />
        <path d="M30.48 16H32v6.09h-1.52Z" />
        <path d="M28.95 22.09h1.53v3.05h-1.53Z" />
        <path d="M27.43 25.14h1.52v3.05h-1.52Z" />
        <path d="M25.91 28.19h1.52v1.52h-1.52Z" />
        <path d="M22.86 29.71h3.05v1.53h-3.05Z" />
        <path d="M21.34 28.19h1.52v1.52h-1.52Z" />
        <path d="M19.81 25.14h1.53v3.05h-1.53Z" />
        <path d="M18.29 22.09h1.52v3.05h-1.52Z" />
        <path d="M16.76 29.71h3.05v1.53h-3.05Z" />
        <path d="M16.76 16h1.53v6.09h-1.53Z" />
        <path d="M13.72 5.33h1.52v6.1h-1.52Z" />
        <path d="M12.19 29.71h3.05v1.53h-3.05Z" />
        <path d="M12.19 11.43h1.53v3.04h-1.53Z" />
        <path d="M12.19 3.81h1.53v1.52h-1.53Z" />
        <path d="M10.67 14.47h1.52v3.05h-1.52Z" />
        <path d="M10.67 2.28h1.52v1.53h-1.52Z" />
        <path d="M9.14 17.52h1.53v1.53H9.14Z" />
        <path d="M7.62 29.71h3.05v1.53H7.62Z" />
        <path d="m9.14 3.81 -3.04 0 0 1.52 -1.53 0 0 3.05 1.53 0 0 1.52 3.04 0 0 -1.52 1.53 0 0 -3.05 -1.53 0 0 -1.52z" />
        <path d="M6.1 26.67h1.52v3.04H6.1Z" />
        <path d="M6.1 22.09h1.52v3.05H6.1Z" />
        <path d="M6.1 19.05h3.04v1.52H6.1Z" />
        <path d="M4.57 0.76h6.1v1.52h-6.1Z" />
        <path d="M4.57 17.52H6.1v1.53H4.57Z" />
        <path d="M3.05 14.47h1.52v3.05H3.05Z" />
        <path d="M3.05 2.28h1.52v1.53H3.05Z" />
        <path d="M1.53 11.43h1.52v3.04H1.53Z" />
        <path d="M1.53 3.81h1.52v1.52H1.53Z" />
        <path d="M0 5.33h1.53v6.1H0Z" />
      </g>
    </svg>
  );
}
