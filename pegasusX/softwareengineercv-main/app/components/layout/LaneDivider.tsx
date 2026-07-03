type LaneDividerProps = {
  index: string;
  label: string;
};

export default function LaneDivider({ index, label }: LaneDividerProps) {
  return (
    <div className="lane-divider" role="presentation" aria-hidden>
      <span className="lane-divider__label">
        <strong>{index}</strong>
        <span aria-hidden> · </span>
        {label}
      </span>
    </div>
  );
}
