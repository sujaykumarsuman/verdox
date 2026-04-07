import type {
  WorkflowConfig,
  WorkflowService,
  WorkflowStep,
  WorkflowMatrix,
  WorkflowConcurrency,
} from "@/types/test";

export type {
  WorkflowConfig,
  WorkflowService,
  WorkflowStep,
  WorkflowMatrix,
  WorkflowConcurrency,
};

/**
 * Generates a GitHub Actions workflow YAML string from a WorkflowConfig.
 * Mirrors the backend's GenerateWorkflowYAML in Go.
 * suiteName is used in the workflow `name:` field shown in the GitHub Actions UI.
 */
export function generateWorkflowYaml(
  config: WorkflowConfig,
  testCommand: string,
  suiteName?: string
): string {
  const lines: string[] = [];

  const runnerOS =
    config.custom_runner || config.runner_os || "ubuntu-latest";

  // Header — name shown in GitHub Actions UI
  const workflowName = suiteName
    ? `Verdox Test Run: ${suiteName}`
    : "Verdox Test Runner";
  lines.push(`name: '${workflowName}'`);
  lines.push("on:");
  lines.push("  workflow_dispatch:");
  lines.push("    inputs:");
  lines.push("      verdox_run_id:");
  lines.push("        description: 'Verdox test run ID'");
  lines.push("        required: true");
  lines.push("      branch:");
  lines.push("        description: 'Branch to test'");
  lines.push("        required: true");
  lines.push("      commit_hash:");
  lines.push("        description: 'Commit hash to test'");
  lines.push("        required: true");
  lines.push("      callback_url:");
  lines.push("        description: 'Webhook callback URL'");
  lines.push("        required: false");
  lines.push("      test_command:");
  lines.push("        description: 'Test command to execute'");
  lines.push("        required: false");
  lines.push(
    `        default: '${testCommand || "make test"}'`
  );
  lines.push("");

  // Concurrency
  if (config.concurrency) {
    lines.push("concurrency:");
    lines.push(`  group: ${config.concurrency.group}`);
    lines.push(
      `  cancel-in-progress: ${config.concurrency.cancel_in_progress}`
    );
    lines.push("");
  }

  // Global env
  const envEntries = Object.entries(config.env_vars || {});
  if (envEntries.length > 0) {
    lines.push("env:");
    for (const [k, v] of envEntries) {
      lines.push(`  ${k}: '${v}'`);
    }
    lines.push("");
  }

  lines.push("jobs:");
  lines.push("  test:");
  lines.push(`    runs-on: ${runnerOS}`);

  // Matrix strategy
  if (config.matrix && Object.keys(config.matrix.dimensions).length > 0) {
    lines.push("    strategy:");
    lines.push(`      fail-fast: ${config.matrix.fail_fast}`);
    lines.push("      matrix:");
    for (const [key, values] of Object.entries(config.matrix.dimensions)) {
      const formatted = values
        .map((v) => `'${v.trim()}'`)
        .join(", ");
      lines.push(`        ${key}: [${formatted}]`);
    }
  }

  // Services
  if (config.services && config.services.length > 0) {
    lines.push("    services:");
    for (const svc of config.services) {
      lines.push(`      ${svc.name}:`);
      lines.push(`        image: ${svc.image}`);
      if (svc.ports && svc.ports.length > 0) {
        lines.push("        ports:");
        for (const port of svc.ports) {
          lines.push(`          - '${port}'`);
        }
      }
      if (svc.env && Object.keys(svc.env).length > 0) {
        lines.push("        env:");
        for (const [k, v] of Object.entries(svc.env)) {
          lines.push(`          ${k}: '${v}'`);
        }
      }
      if (svc.ports && svc.ports.length > 0) {
        lines.push("        options: >-");
        lines.push('          --health-cmd "exit 0"');
        lines.push("          --health-interval 10s");
        lines.push("          --health-timeout 5s");
        lines.push("          --health-retries 5");
      }
    }
  }

  // Steps
  lines.push("    steps:");

  // Checkout
  lines.push("      - name: Checkout");
  lines.push("        uses: actions/checkout@v4");
  lines.push("        with:");
  lines.push(
    "          ref: ${{ github.event.inputs.commit_hash }}"
  );
  lines.push("");

  // User-defined setup steps
  if (config.setup_steps) {
    for (const step of config.setup_steps) {
      if (step.uses) {
        lines.push(`      - name: ${step.name}`);
        lines.push(`        uses: ${step.uses}`);
        if (step.with && Object.keys(step.with).length > 0) {
          lines.push("        with:");
          for (const [k, v] of Object.entries(step.with)) {
            lines.push(`          ${k}: '${v}'`);
          }
        }
      } else if (step.run) {
        lines.push(`      - name: ${step.name}`);
        lines.push("        run: |");
        for (const line of step.run.split("\n")) {
          lines.push(`          ${line}`);
        }
      }
      lines.push("");
    }
  }

  // Test execution step
  lines.push("      - name: Run tests");
  lines.push("        id: tests");
  lines.push("        run: |");
  lines.push("          set +e");
  lines.push(
    "          ${{ github.event.inputs.test_command }} 2>&1 | tee test-output.log"
  );
  lines.push('          echo "exit_code=$?" >> $GITHUB_OUTPUT');
  lines.push("        continue-on-error: true");
  lines.push("");

  // Results JSON
  lines.push("      - name: Generate results JSON");
  lines.push("        if: always()");
  lines.push("        run: |");
  lines.push("          cat > verdox-results.json << 'RESULTS_EOF'");
  lines.push("          {");
  lines.push(
    '            "verdox_run_id": "${{ github.event.inputs.verdox_run_id }}",'
  );
  lines.push(
    '            "status": "${{ steps.tests.outcome }}",'
  );
  lines.push(
    '            "exit_code": ${{ steps.tests.outputs.exit_code || 1 }}'
  );
  lines.push("          }");
  lines.push("          RESULTS_EOF");
  lines.push("");

  // Upload artifact
  lines.push("      - name: Upload results artifact");
  lines.push("        if: always()");
  lines.push("        uses: actions/upload-artifact@v4");
  lines.push("        with:");
  lines.push("          name: verdox-results");
  lines.push("          path: |");
  lines.push("            verdox-results.json");
  lines.push("            test-output.log");
  lines.push("          retention-days: 7");
  lines.push("");

  // Callback
  lines.push("      - name: Callback to Verdox");
  lines.push(
    "        if: always() && github.event.inputs.callback_url != ''"
  );
  lines.push("        run: |");
  lines.push(
    '          curl -s -X POST "${{ github.event.inputs.callback_url }}" \\'
  );
  lines.push(
    '            -H "Content-Type: application/json" \\'
  );
  lines.push("            -d @verdox-results.json || true");

  return lines.join("\n");
}
