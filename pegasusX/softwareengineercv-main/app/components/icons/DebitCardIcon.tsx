type DebitCardIconProps = {
  className?: string;
  size?: number;
  title?: string;
};

/** Pixel debit card — pay-at-delivery collection (matches Streamline grid). */
export default function DebitCardIcon({ className = '', size = 20, title }: DebitCardIconProps) {
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
        <path d="M0 9.14h32v13.72H0Z" />
        <path d="M0 10.67h32v3.05H0Z" />
        <path d="M3.05 16.76h7.62v6.1H3.05Z" />
        <path d="M4.57 18.29h4.58v3.05H4.57Z" />
        <path d="M13.72 21.34h12.19v1.52H13.72Z" />
        <path d="M19.81 19.81h6.1v1.53h-6.1Z" />
        <path d="M22.86 18.29h3.05v1.52h-3.05Z" />
        <path d="M1.52 7.62h28.96v1.52H1.52Z" />
        <path d="M0 6.1h1.52v19.81H0Z" />
        <path d="M30.48 6.1H32v19.81h-1.52Z" />
        <path d="M1.52 24.38h28.96v1.52H1.52Z" />
      </g>
    </svg>
  );
}
