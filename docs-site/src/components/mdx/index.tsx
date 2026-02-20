import type { MDXComponents } from "mdx/types";
import { Callout } from "@/components/Callout";
import { Tabs, Tab } from "@/components/Tabs";
import { CodeBlock } from "@/components/CodeBlock";

export const mdxComponents: MDXComponents = {
  h1: (props) => (
    <h1 className="text-3xl font-bold mt-2 mb-6 text-text-primary" {...props} />
  ),
  h2: (props) => (
    <h2
      className="text-2xl font-semibold mt-10 mb-4 pb-2 border-b border-border text-text-primary scroll-mt-20"
      {...props}
    />
  ),
  h3: (props) => (
    <h3 className="text-xl font-semibold mt-8 mb-3 text-text-primary scroll-mt-20" {...props} />
  ),
  h4: (props) => (
    <h4 className="text-lg font-medium mt-6 mb-2 text-text-primary scroll-mt-20" {...props} />
  ),
  p: (props) => (
    <p className="leading-7 mb-4 text-text-secondary" {...props} />
  ),
  a: (props) => (
    <a
      className="text-accent hover:text-accent-hover underline underline-offset-4 transition-colors"
      {...props}
    />
  ),
  ul: (props) => (
    <ul className="list-disc pl-6 mb-4 space-y-1 text-text-secondary" {...props} />
  ),
  ol: (props) => (
    <ol className="list-decimal pl-6 mb-4 space-y-1 text-text-secondary" {...props} />
  ),
  li: (props) => <li className="leading-7" {...props} />,
  blockquote: (props) => (
    <blockquote
      className="border-l-4 border-accent pl-4 italic text-text-secondary my-4"
      {...props}
    />
  ),
  table: (props) => (
    <div className="overflow-x-auto mb-6 rounded-lg border border-border">
      <table className="w-full text-sm" {...props} />
    </div>
  ),
  thead: (props) => <thead className="bg-surface-tertiary" {...props} />,
  th: (props) => (
    <th
      className="px-4 py-2.5 text-left font-semibold text-text-primary border-b border-border"
      {...props}
    />
  ),
  td: (props) => (
    <td className="px-4 py-2.5 text-text-secondary border-b border-border" {...props} />
  ),
  hr: (props) => <hr className="border-border my-8" {...props} />,
  pre: CodeBlock,
  Callout,
  Tabs,
  Tab,
};
