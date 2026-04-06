"use client";

import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-bg-primary">
      <div className="text-center px-4">
        <AlertTriangle className="h-12 w-12 text-danger mx-auto mb-4" />
        <h1 className="text-[24px] font-semibold text-text-primary mb-2">
          Something went wrong
        </h1>
        <p className="text-[14px] text-text-secondary mb-8 max-w-sm mx-auto">
          {error.message || "An unexpected error occurred. Please try again."}
        </p>
        <Button onClick={reset}>Try again</Button>
      </div>
    </div>
  );
}
