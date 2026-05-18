interface EmptyStateProps {
  title: string;
  message: string;
}

export function EmptyState({ message, title }: EmptyStateProps) {
  return (
    <div className="emptyState">
      <h2>{title}</h2>
      <p>{message}</p>
    </div>
  );
}
