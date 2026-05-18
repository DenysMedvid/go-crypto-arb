import type { ReactNode } from 'react';

interface SummaryCardProps {
  title: string;
  value: ReactNode;
  detail?: ReactNode;
  tone?: 'ok' | 'warning' | 'error' | 'neutral';
}

export function SummaryCard({ detail, title, tone = 'neutral', value }: SummaryCardProps) {
  return (
    <section className={`summaryCard ${tone}`} aria-label={title}>
      <div className="summaryTitle">{title}</div>
      <div className="summaryValue">{value}</div>
      {detail ? <div className="summaryDetail">{detail}</div> : null}
    </section>
  );
}
