interface StatusBadgeProps {
  label: string;
  tone?: 'ok' | 'warning' | 'error' | 'muted';
}

export function StatusBadge({ label, tone = 'muted' }: StatusBadgeProps) {
  return <span className={`statusBadge ${tone}`}>{label}</span>;
}
