type GlitchTextProps = {
  text: string;
  className?: string;
  as?: "h1" | "h2" | "span" | "p";
};

export function GlitchText({ text, className = "", as: Tag = "span" }: GlitchTextProps) {
  return (
    <Tag
      data-text={text}
      className={`void-glitch relative mx-auto inline-block cursor-default font-black uppercase select-none text-white ${className}`}
    >
      {text}
    </Tag>
  );
}
