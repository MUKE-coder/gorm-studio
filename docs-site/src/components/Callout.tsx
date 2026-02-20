const icons = {
  info: (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10" />
      <line x1="12" y1="16" x2="12" y2="12" />
      <line x1="12" y1="8" x2="12.01" y2="8" />
    </svg>
  ),
  warning: (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
      <line x1="12" y1="9" x2="12" y2="13" />
      <line x1="12" y1="17" x2="12.01" y2="17" />
    </svg>
  ),
  danger: (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polygon points="7.86 2 16.14 2 22 7.86 22 16.14 16.14 22 7.86 22 2 16.14 2 7.86 7.86 2" />
      <line x1="12" y1="8" x2="12" y2="12" />
      <line x1="12" y1="16" x2="12.01" y2="16" />
    </svg>
  ),
};

const styles = {
  info: "border-info/30 bg-info/5 text-info",
  warning: "border-warning/30 bg-warning/5 text-warning",
  danger: "border-danger/30 bg-danger/5 text-danger",
};

export function Callout({
  type = "info",
  children,
}: {
  type?: "info" | "warning" | "danger";
  children: React.ReactNode;
}) {
  return (
    <div className={`flex gap-3 p-4 my-4 rounded-lg border ${styles[type]}`}>
      <div className="shrink-0 mt-0.5">{icons[type]}</div>
      <div className="text-sm [&>p]:mb-0 [&>p]:text-current">{children}</div>
    </div>
  );
}
