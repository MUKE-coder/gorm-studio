"use client";

import { useState, Children } from "react";

export function Tabs({
  items,
  children,
}: {
  items: string[];
  children: React.ReactNode;
}) {
  const [active, setActive] = useState(0);
  const tabs = Children.toArray(children);

  return (
    <div className="my-4">
      <div className="flex border-b border-border">
        {items.map((item, i) => (
          <button
            key={item}
            onClick={() => setActive(i)}
            className={`px-4 py-2 text-sm font-medium transition-colors border-b-2 -mb-px ${
              active === i
                ? "border-accent text-accent"
                : "border-transparent text-text-muted hover:text-text-secondary"
            }`}
          >
            {item}
          </button>
        ))}
      </div>
      <div className="mt-4">{tabs[active]}</div>
    </div>
  );
}

export function Tab({ children }: { children: React.ReactNode }) {
  return <div>{children}</div>;
}
