import type { ReactNode } from 'react';

interface PageHeaderProps {
  title: string;
  subtitle?: ReactNode;
}

export function PageHeader({ subtitle, title }: PageHeaderProps) {
  return (
    <div className="pageHeader">
      <h1>{title}</h1>
      {subtitle ? <p>{subtitle}</p> : null}
    </div>
  );
}
