"use client";

import Link from "next/link";
import { CopyButton } from "./CopyButton";

const installCmd = "go get github.com/MUKE-coder/gorm-studio/studio";

export function Hero() {
  return (
    <section className="relative overflow-hidden">
      <div className="absolute inset-0 bg-gradient-to-b from-accent/5 to-transparent pointer-events-none" />
      <div className="max-w-4xl mx-auto px-6 pt-24 pb-20 text-center relative">
        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-accent-muted text-accent text-sm font-medium mb-6">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
          </svg>
          Open Source Database Browser for Go
        </div>
        <h1 className="text-5xl md:text-6xl font-bold text-text-primary mb-6 leading-tight">
          Visual Database Browser{" "}
          <span className="text-accent">for GORM</span>
        </h1>
        <p className="text-lg text-text-secondary max-w-2xl mx-auto mb-8 leading-relaxed">
          A Prisma Studio-like database management UI that mounts directly into your Go application.
          Browse schemas, manage data, run SQL, export ERD diagrams, and import data &mdash; all from a
          single <code className="text-accent bg-accent-muted px-1.5 py-0.5 rounded text-sm font-mono">studio.Mount()</code> call.
        </p>

        <div className="flex flex-col sm:flex-row items-center justify-center gap-3 mb-10">
          <Link
            href="/docs/getting-started/introduction"
            className="px-6 py-3 rounded-lg bg-accent hover:bg-accent-hover text-accent-foreground font-medium text-sm transition-colors"
          >
            Get Started
          </Link>
          <a
            href="https://github.com/MUKE-coder/gorm-studio"
            target="_blank"
            rel="noopener noreferrer"
            className="px-6 py-3 rounded-lg border border-border hover:bg-surface-hover text-text-primary font-medium text-sm transition-colors"
          >
            View on GitHub
          </a>
        </div>

        <div className="inline-flex items-center gap-2 bg-surface-secondary border border-border rounded-lg px-4 py-2.5 font-mono text-sm text-text-secondary">
          <span className="text-text-muted">$</span>
          <span>{installCmd}</span>
          <CopyButton text={installCmd} />
        </div>
      </div>
    </section>
  );
}
