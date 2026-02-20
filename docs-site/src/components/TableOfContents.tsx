"use client";

import { useEffect, useState } from "react";
import type { TocItem } from "@/lib/toc";

export function TableOfContents({ items }: { items: TocItem[] }) {
  const [activeId, setActiveId] = useState<string>("");

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setActiveId(entry.target.id);
          }
        }
      },
      { rootMargin: "-80px 0px -80% 0px" }
    );

    for (const item of items) {
      const el = document.getElementById(item.id);
      if (el) observer.observe(el);
    }

    return () => observer.disconnect();
  }, [items]);

  if (items.length === 0) return null;

  return (
    <aside className="hidden xl:block w-56 shrink-0">
      <div className="sticky top-24">
        <div className="text-xs font-semibold uppercase tracking-wider text-text-muted mb-3">
          On this page
        </div>
        <nav className="space-y-1">
          {items.map((item) => (
            <a
              key={item.id}
              href={`#${item.id}`}
              className={`block text-sm py-1 transition-colors ${
                item.level === 3 ? "pl-4" : item.level === 4 ? "pl-8" : ""
              } ${
                activeId === item.id
                  ? "text-accent font-medium"
                  : "text-text-muted hover:text-text-secondary"
              }`}
            >
              {item.title}
            </a>
          ))}
        </nav>
      </div>
    </aside>
  );
}
