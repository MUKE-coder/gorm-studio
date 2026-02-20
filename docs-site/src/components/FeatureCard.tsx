export function FeatureCard({
  icon,
  title,
  description,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
}) {
  return (
    <div className="p-6 rounded-xl border border-border bg-surface-secondary hover:border-accent/30 transition-colors group">
      <div className="w-10 h-10 rounded-lg bg-accent-muted flex items-center justify-center text-accent mb-4 group-hover:bg-accent group-hover:text-accent-foreground transition-colors">
        {icon}
      </div>
      <h3 className="font-semibold text-text-primary mb-2">{title}</h3>
      <p className="text-sm text-text-secondary leading-relaxed">{description}</p>
    </div>
  );
}
