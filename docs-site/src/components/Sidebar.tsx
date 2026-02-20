"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { navigation } from "@/lib/navigation";
import { Logo } from "./Logo";

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="hidden lg:block fixed top-0 left-0 w-72 h-screen overflow-y-auto border-r border-border bg-surface-primary">
      <div className="p-5 border-b border-border">
        <Logo />
      </div>
      <nav className="p-4">
        {navigation.map((group) => (
          <div key={group.title} className="mb-6">
            <div className="text-xs font-semibold uppercase tracking-wider text-text-muted mb-2 px-3">
              {group.title}
            </div>
            {group.items.map((item) => {
              const isActive = pathname === item.href;
              return (
                <Link
                  key={item.slug}
                  href={item.href}
                  className={`block px-3 py-1.5 text-sm rounded-md transition-colors mb-0.5 ${
                    isActive
                      ? "text-accent bg-accent-muted font-medium border-l-2 border-accent"
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
    </aside>
  );
}
