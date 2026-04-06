"use client";

import { Sparkles, Plus } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardBody } from "@/components/ui/card";
import type { DiscoverySuggestion } from "@/types/test";

interface DiscoveryPanelProps {
  suggestions: DiscoverySuggestion[];
  onApply: (suggestion: DiscoverySuggestion) => void;
  onDismiss: () => void;
}

export function DiscoveryPanel({
  suggestions,
  onApply,
  onDismiss,
}: DiscoveryPanelProps) {
  if (suggestions.length === 0) return null;

  return (
    <Card className="mb-6 border-accent/30 bg-[var(--accent-subtle)]/30">
      <CardBody>
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Sparkles className="h-4 w-4 text-accent" />
            <h4 className="text-sm font-semibold text-text-primary">
              AI-Discovered Test Suites
            </h4>
          </div>
          <button
            onClick={onDismiss}
            className="text-xs text-text-secondary hover:text-text-primary"
          >
            Dismiss
          </button>
        </div>

        <div className="space-y-3">
          {suggestions.map((suggestion, idx) => (
            <div
              key={idx}
              className="flex items-start justify-between p-3 rounded-[6px] bg-bg-secondary border border-[var(--border)]"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-sm font-medium text-text-primary">
                    {suggestion.name}
                  </span>
                  <Badge variant="neutral">{suggestion.type}</Badge>
                  <Badge variant={suggestion.execution_mode === "container" ? "info" : "warning"}>
                    {suggestion.execution_mode === "container" ? "Container" : "GHA"}
                  </Badge>
                </div>
                <p className="text-[12px] text-text-secondary">
                  {suggestion.reasoning}
                </p>
                {suggestion.test_command && (
                  <code className="text-[11px] font-mono text-text-secondary mt-1 block">
                    {suggestion.test_command}
                  </code>
                )}
              </div>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => onApply(suggestion)}
                className="ml-3 shrink-0"
              >
                <Plus className="h-3 w-3 mr-1" />
                Apply
              </Button>
            </div>
          ))}
        </div>
      </CardBody>
    </Card>
  );
}
