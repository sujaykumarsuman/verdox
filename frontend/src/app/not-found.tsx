import Link from "next/link";

export default function NotFound() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-bg-primary">
      <div className="text-center px-4">
        <p className="text-[72px] font-display text-accent leading-none mb-4">
          404
        </p>
        <h1 className="text-[24px] font-semibold text-text-primary mb-2">
          Page not found
        </h1>
        <p className="text-[14px] text-text-secondary mb-8 max-w-sm mx-auto">
          The page you&apos;re looking for doesn&apos;t exist or has been moved.
        </p>
        <Link
          href="/dashboard"
          className="inline-flex items-center justify-center h-9 px-4 rounded-[6px] bg-accent text-white text-[14px] font-medium hover:bg-accent-light transition-colors"
        >
          Back to Dashboard
        </Link>
      </div>
    </div>
  );
}
