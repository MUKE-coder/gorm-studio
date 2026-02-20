import Link from "next/link";
import { Navbar } from "@/components/Navbar";

export default function NotFound() {
  return (
    <div className="min-h-screen">
      <Navbar />
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-4">
        <h1 className="text-6xl font-bold text-text-muted">404</h1>
        <p className="text-text-secondary text-lg">Page not found</p>
        <Link
          href="/"
          className="mt-4 px-5 py-2.5 rounded-lg bg-accent hover:bg-accent-hover text-accent-foreground text-sm font-medium transition-colors"
        >
          Go Home
        </Link>
      </div>
    </div>
  );
}
