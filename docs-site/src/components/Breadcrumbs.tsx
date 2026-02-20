import Link from "next/link";

interface BreadcrumbItem {
  title: string;
  href: string;
}

export function Breadcrumbs({ items }: { items: BreadcrumbItem[] }) {
  return (
    <nav className="flex items-center gap-1.5 text-sm text-text-muted mb-6">
      {items.map((item, i) => (
        <span key={item.href} className="flex items-center gap-1.5">
          {i > 0 && (
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="9 18 15 12 9 6" />
            </svg>
          )}
          {i === items.length - 1 ? (
            <span className="text-text-secondary">{item.title}</span>
          ) : (
            <Link href={item.href} className="hover:text-text-secondary transition-colors">
              {item.title}
            </Link>
          )}
        </span>
      ))}
    </nav>
  );
}
