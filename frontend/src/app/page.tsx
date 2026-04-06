import Link from "next/link";
import { Button } from "@/components/ui/button";

export default function LandingPage() {
  return (
    <main className="min-h-screen flex flex-col">
      {/* Hero */}
      <section className="flex-1 flex items-center justify-center px-4">
        <div className="max-w-2xl text-center">
          <h1 className="font-display text-[36px] leading-[44px] tracking-[-0.02em] text-text-primary mb-4">
            Test orchestration,
            <br />
            under your control.
          </h1>
          <p className="font-body text-[18px] leading-[28px] text-text-secondary mb-8 max-w-lg mx-auto">
            Verdox is a self-hosted platform for managing, running, and monitoring
            your test suites. Connect your repos, trigger runs, and get results
            &mdash; all in one place.
          </p>
          <div className="flex items-center justify-center gap-4">
            <Link href="/login">
              <Button variant="secondary" size="lg">
                Login
              </Button>
            </Link>
            <Link href="/signup">
              <Button variant="primary" size="lg">
                Sign Up
              </Button>
            </Link>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="pb-16 px-4">
        <div className="max-w-4xl mx-auto grid grid-cols-1 md:grid-cols-3 gap-6">
          <FeatureCard
            title="Self-Hosted"
            description="Run on your own infrastructure. Your code never leaves your network."
          />
          <FeatureCard
            title="Team-First"
            description="Organize repos by team, manage access, and collaborate on test results."
          />
          <FeatureCard
            title="Docker-Powered"
            description="Each test run executes in an isolated Docker container for reproducibility."
          />
        </div>
      </section>

      {/* Footer */}
      <footer className="py-6 text-center text-[12px] text-text-secondary border-t">
        &copy; {new Date().getFullYear()} Verdox
      </footer>
    </main>
  );
}

function FeatureCard({ title, description }: { title: string; description: string }) {
  return (
    <div className="rounded-[8px] border bg-bg-secondary p-6 shadow-[var(--shadow-card)]">
      <h3 className="font-body text-[20px] leading-[28px] font-semibold text-text-primary mb-2">
        {title}
      </h3>
      <p className="text-[14px] leading-[20px] text-text-secondary">{description}</p>
    </div>
  );
}
