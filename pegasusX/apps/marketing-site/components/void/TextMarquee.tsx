"use client";

type TextMarqueeProps = {
  items: string[];
  separator?: string;
  className?: string;
  speed?: "slow" | "normal" | "fast";
};

export function TextMarquee({
  items,
  separator = " ",
  className = "",
  speed = "normal",
}: TextMarqueeProps) {
  const track = [...items, ...items];
  const duration = speed === "slow" ? "55s" : speed === "fast" ? "25s" : "38s";

  return (
    <div className={`void-marquee overflow-hidden ${className}`} aria-hidden>
      <div
        className="void-marquee__track flex w-max gap-8"
        style={{ animationDuration: duration }}
      >
        {track.map((item, i) => (
          <span key={`${item}-${i}`} className="void-marquee__item shrink-0 font-black uppercase tracking-[0.2em]">
            {item}
            {separator ? <span className="mx-4 opacity-40">{separator}</span> : null}
          </span>
        ))}
      </div>
    </div>
  );
}
