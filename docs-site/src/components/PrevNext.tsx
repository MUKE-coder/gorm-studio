import Link from "next/link";
import type { NavItem } from "@/lib/navigation";

export function PrevNext({ prev, next }: { prev: NavItem | null; next: NavItem | null }) {
  return (
    <div className="flex items-stretch gap-4 mt-12 pt-8 border-t border-border">
      {prev ? (
        <Link
          href={prev.href}
          className="flex-1 group p-4 rounded-lg border border-border hover:border-accent/50 transition-colors"
        >
          <div className="text-xs text-text-muted mb-1">Previous</div>
          <div className="text-sm font-medium text-text-primary group-hover:text-accent transition-colors">
            &larr; {prev.title}
          </div>
        </Link>
      ) : (
        <div className="flex-1" />
      )}
      {next ? (
        <Link
          href={next.href}
          className="flex-1 group p-4 rounded-lg border border-border hover:border-accent/50 transition-colors text-right"
        >
          <div className="text-xs text-text-muted mb-1">Next</div>
          <div className="text-sm font-medium text-text-primary group-hover:text-accent transition-colors">
            {next.title} &rarr;
          </div>
        </Link>
      ) : (
        <div className="flex-1" />
      )}
    </div>
  );
}
