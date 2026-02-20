import { Navbar } from "@/components/Navbar";
import { Sidebar } from "@/components/Sidebar";

export default function DocsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen">
      <div className="lg:hidden">
        <Navbar />
      </div>
      <div className="flex">
        <Sidebar />
        <div className="flex-1 lg:pl-72">
          <main className="max-w-4xl mx-auto px-6 py-10">
            {children}
          </main>
        </div>
      </div>
    </div>
  );
}
