import { Navbar } from "@/components/Navbar";
import { Hero } from "@/components/Hero";
import { FeaturesGrid } from "@/components/FeaturesGrid";
import { CodeExample } from "@/components/CodeExample";
import { Footer } from "@/components/Footer";

export default function Home() {
  return (
    <div className="min-h-screen">
      <Navbar />
      <Hero />
      <FeaturesGrid />
      <CodeExample />
      <Footer />
    </div>
  );
}
