"use client";

import { useRef } from "react";
import { CopyButton } from "./CopyButton";

export function CodeBlock(props: React.ComponentProps<"pre">) {
  const ref = useRef<HTMLPreElement>(null);

  const getCodeText = () => {
    return ref.current?.querySelector("code")?.textContent || "";
  };

  return (
    <div className="relative group">
      <pre ref={ref} {...props} />
      <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
        <CopyButton text={getCodeText()} />
      </div>
    </div>
  );
}
