type BusinessBagIconProps = {
  className?: string;
  size?: number;
  title?: string;
};

/** Streamline pixel shopping bag — checkout / retailer cart. */
export default function BusinessBagIcon({ className = '', size = 20, title }: BusinessBagIconProps) {
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
        <path d="M30.47 10.67H32v15.24h-1.53Z" />
        <path d="M9.14 25.91h21.33v1.52H9.14Z" />
        <path d="M27.43 13.72h1.52v3.04h-1.52Z" />
        <path d="m24.38 13.72 3.05 0 0 -1.53 -6.1 0 0 1.53 -1.52 0 0 3.04 1.52 0 0 1.53 6.1 0 0 -1.53 -3.05 0 0 -3.04z" />
        <path d="m9.14 9.15 0 1.52 21.33 0 0 -1.52 -4.57 0 0 -3.05 -1.52 0 0 3.05 -15.24 0z" />
        <path d="M10.66 22.86h9.15v1.52h-9.15Z" />
        <path d="M10.66 19.81h4.58v1.53h-4.58Z" />
        <path d="m9.14 10.67 -1.52 0 0 10.67 -6.1 0 0 1.52 6.1 0 0 3.05 1.52 0 0 -15.24z" />
        <path d="M1.52 4.57h22.86V6.1H1.52Z" />
        <path d="M0 6.1h1.52v15.24H0Z" />
      </g>
    </svg>
  );
}
