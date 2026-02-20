"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { navigation } from "@/lib/navigation";

export function MobileNav({ open, onClose }: { open: boolean; onClose: () => void }) {
  const pathname = usePathname();

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 md:hidden">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="fixed top-0 left-0 bottom-0 w-72 bg-surface-primary border-r border-border overflow-y-auto">
        <div className="p-4 border-b border-border flex items-center justify-between">
          <span className="font-semibold text-text-primary">Navigation</span>
          <button
            onClick={onClose}
            className="p-1.5 rounded-md hover:bg-surface-hover text-text-muted"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
        <nav className="p-4">
          {navigation.map((group) => (
            <div key={group.title} className="mb-6">
              <div className="text-xs font-semibold uppercase tracking-wider text-text-muted mb-2 px-2">
                {group.title}
              </div>
              {group.items.map((item) => {
                const isActive = pathname === item.href;
                return (
                  <Link
                    key={item.slug}
                    href={item.href}
                    onClick={onClose}
                    className={`block px-2 py-1.5 text-sm rounded-md transition-colors ${
                      isActive
                        ? "text-accent bg-accent-muted font-medium"
                        : "text-text-secondary hover:text-text-primary hover:bg-surface-hover"
                    }`}
                  >
                    {item.title}
                  </Link>
                );
              })}
            </div>
          ))}
        </nav>
      </div>
    </div>
  );
}
