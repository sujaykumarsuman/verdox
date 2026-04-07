export interface WorkflowTemplate {
  name: string;
  label: string;
  description: string;
  yaml: string;
}

// The Python script that each go-test job embeds to parse NDJSON into per-test JSON
const GO_TEST_PARSER = `
          python3 << 'VERDOX_PARSER'
          import json, sys
          tests = {}
          pkg_durations = {}  # package -> total elapsed seconds
          try:
              with open("test-output.json", "r") as f:
                  for line in f:
                      line = line.strip()
                      if not line: continue
                      try: ev = json.loads(line)
                      except: continue
                      test = ev.get("Test", "")
                      pkg = ev.get("Package", "")
                      action = ev.get("Action", "")
                      # Package-level events (no Test field) carry total package duration
                      if not test and pkg and action in ("pass", "fail"):
                          pkg_durations[pkg] = ev.get("Elapsed", 0)
                          continue
                      if not test: continue
                      key = f"{pkg}/{test}"
                      if key not in tests:
                          tests[key] = {"name": test, "package": pkg, "status": "", "duration_s": 0, "output": [], "error": ""}
                      t = tests[key]
                      if action == "output": t["output"].append(ev.get("Output", ""))
                      elif action == "pass":
                          t["status"] = "passed"
                          t["duration_s"] = ev.get("Elapsed", 0)
                      elif action == "fail":
                          t["status"] = "failed"
                          t["duration_s"] = ev.get("Elapsed", 0)
                          t["error"] = "".join(t["output"])[-3000:]
                      elif action == "skip": t["status"] = "skipped"
          except Exception as e:
              print(f"Warning: {e}", file=sys.stderr)
          # Group by package
          packages = {}
          for key, t in tests.items():
              if not t["status"]: continue
              pkg = t["package"]
              if pkg not in packages: packages[pkg] = []
              packages[pkg].append(t)
          result = {"packages": {}, "pkg_durations": pkg_durations}
          for pkg, cases in packages.items():
              result["packages"][pkg] = [{"name": c["name"], "status": c["status"], "duration_seconds": c["duration_s"], "error_message": c["error"]} for c in cases]
          with open("verdox-test-results.json", "w") as f:
              json.dump(result, f)
          total = sum(len(v) for v in result["packages"].values())
          print(f"Verdox: parsed {total} tests across {len(result['packages'])} packages")
          VERDOX_PARSER`;

// The Python script the final verdox-report job uses to build schema.json payload
const REPORT_COLLECTOR_SCRIPT = `
          python3 << 'VERDOX_REPORT'
          import json, os, sys, glob
          from datetime import datetime, timezone

          run_id = os.environ.get("VERDOX_RUN_ID", "")
          repo = os.environ.get("VERDOX_REPO", "")
          branch = os.environ.get("VERDOX_BRANCH", "")
          commit = os.environ.get("VERDOX_COMMIT", "")
          suites_json = os.environ.get("VERDOX_SUITES", "[]")

          suites_config = json.loads(suites_json)
          suites = []
          all_cases = 0
          all_passed = 0
          all_failed = 0
          all_skipped = 0
          all_duration = 0.0

          for sc in suites_config:
              suite_id = sc["id"]
              suite_name = sc["name"]
              suite_type = sc["type"]
              job_result = sc.get("result", "success")
              results_file = sc.get("results_file", "")

              tests = []
              suite_passed = suite_failed = suite_skipped = suite_total = 0
              suite_duration = 0.0

              # Try loading go test -json parsed output
              if results_file and os.path.exists(results_file):
                  with open(results_file) as f:
                      data = json.load(f)
                  pkg_dur = data.get("pkg_durations", {})
                  for pkg, cases in data.get("packages", {}).items():
                      pkg_slug = pkg.replace("/", "-").replace(".", "-")[-60:]
                      pkg_name = pkg.split("/")[-1] if "/" in pkg else pkg
                      # Use package-level duration from go test (more accurate than sum of test durations)
                      pkg_elapsed = pkg_dur.get(pkg, 0)
                      test_cases = []
                      t_passed = t_failed = t_skipped = 0
                      t_duration = 0.0
                      for c in cases:
                          st = c.get("status", "unknown")
                          dur = c.get("duration_seconds", 0)
                          tc = {"case_id": c["name"], "name": c["name"], "status": st, "duration_seconds": dur}
                          if c.get("error_message"): tc["error_message"] = c["error_message"]
                          test_cases.append(tc)
                          if st == "passed": t_passed += 1
                          elif st == "failed": t_failed += 1
                          elif st == "skipped": t_skipped += 1
                          t_duration += dur
                      # Prefer package-level elapsed over sum of individual test durations
                      if pkg_elapsed > t_duration: t_duration = pkg_elapsed
                      t_total = len(test_cases)
                      t_status = "failed" if t_failed > 0 else "passed"
                      t_rate = round(t_passed / t_total * 100, 2) if t_total > 0 else 0
                      tests.append({
                          "test_id": pkg_slug, "name": pkg_name, "package": pkg, "status": t_status,
                          "stats": {"total": t_total, "passed": t_passed, "failed": t_failed, "skipped": t_skipped, "duration_seconds": round(t_duration, 2), "pass_rate": t_rate},
                          "cases": test_cases
                      })
                      suite_passed += t_passed; suite_failed += t_failed; suite_skipped += t_skipped
                      suite_total += t_total; suite_duration += t_duration
              else:
                  # Simple pass/fail job (lint, build) — single test case
                  st = "passed" if job_result == "success" else "failed"
                  tests.append({
                      "test_id": suite_id, "name": suite_name, "package": "", "status": st,
                      "stats": {"total": 1, "passed": 1 if st == "passed" else 0, "failed": 1 if st == "failed" else 0, "skipped": 0, "duration_seconds": 0, "pass_rate": 100 if st == "passed" else 0},
                      "cases": [{"case_id": suite_id, "name": suite_name, "status": st, "duration_seconds": 0}]
                  })
                  suite_total = 1
                  if st == "passed": suite_passed = 1
                  else: suite_failed = 1

              suite_status = "failed" if suite_failed > 0 else "passed"
              suite_rate = round(suite_passed / suite_total * 100, 2) if suite_total > 0 else 0
              suites.append({
                  "job_id": suite_id, "name": suite_name, "type": suite_type, "status": suite_status,
                  "stats": {"total": suite_total, "passed": suite_passed, "failed": suite_failed, "skipped": suite_skipped, "duration_seconds": round(suite_duration, 2), "pass_rate": suite_rate},
                  "tests": tests
              })
              all_cases += suite_total; all_passed += suite_passed; all_failed += suite_failed; all_skipped += suite_skipped; all_duration += suite_duration

          all_rate = round(all_passed / all_cases * 100, 2) if all_cases > 0 else 0
          payload = {
              "repo": repo, "run_id": run_id, "branch": branch, "commit_sha": commit,
              "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
              "summary": {"total_jobs": len(suites), "total_tests": sum(len(s["tests"]) for s in suites), "total_cases": all_cases, "passed": all_passed, "failed": all_failed, "skipped": all_skipped, "duration_seconds": round(all_duration, 2), "pass_rate": all_rate},
              "jobs": suites
          }
          with open("verdox-results.json", "w") as f:
              json.dump(payload, f, indent=2)
          print(f"Verdox report: {len(suites)} suites, {all_cases} cases ({all_passed}P/{all_failed}F/{all_skipped}S)")
          VERDOX_REPORT`;

const VERDOX_HEADER = `# Verdox CI Workflow
# This workflow mirrors your CI and reports structured results to Verdox.
#
# Pattern: each CI job runs independently, then a final verdox-report job
# collects all outcomes and builds a hierarchical test report (schema.json).
#
# REQUIRED: workflow_dispatch trigger with verdox inputs
# REQUIRED: verdox-report job at the end
`;

export const workflowTemplates: WorkflowTemplate[] = [
  {
    name: "blank",
    label: "Multi-Job CI (Default)",
    description: "Multi-job workflow with per-job results collection — mirrors your existing CI",
    yaml: `${VERDOX_HEADER}
name: "verdox: CI"

on:
  workflow_dispatch:
    inputs:
      verdox_run_id:
        description: 'Verdox test run ID'
        required: true
      branch:
        description: 'Branch to test'
        required: true
      commit_hash:
        description: 'Commit hash to test'
        required: true
      callback_url:
        description: 'Webhook callback URL'
        required: false

# ============================================
# CI JOBS — mirror your existing CI here
# For go test jobs: use -json flag and upload verdox-test-results.json artifact
# For lint/build jobs: just let them pass or fail
# ============================================

jobs:

  # --- Example: Go test job (produces per-test results) ---
  backend-test:
    name: Backend Test
    runs-on: ubuntu-latest
    # services:
    #   postgres:
    #     image: postgres:16-alpine
    #     env: { POSTGRES_USER: app, POSTGRES_PASSWORD: app, POSTGRES_DB: app_test }
    #     ports: ["5432:5432"]
    #     options: --health-cmd "pg_isready" --health-interval 5s --health-timeout 3s --health-retries 5
    steps:
      - uses: actions/checkout@v4
        with:
          ref: \${{ github.event.inputs.branch }}
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Run tests
        run: |
          set +e
          go test -json -race -count=1 ./... 2>&1 | tee test-output.json
          exit 0
      - name: Parse test results
        if: always()
        run: |${GO_TEST_PARSER}
      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: backend-test-results
          path: verdox-test-results.json

  # --- Example: Lint job (simple pass/fail) ---
  # backend-lint:
  #   name: Backend Lint
  #   runs-on: ubuntu-latest
  #   steps:
  #     - uses: actions/checkout@v4
  #       with:
  #         ref: \${{ github.event.inputs.branch }}
  #     - uses: actions/setup-go@v5
  #       with:
  #         go-version-file: go.mod
  #     - name: Lint
  #       run: golangci-lint run ./...

  # ============================================
  # VERDOX REPORT — collects results from all jobs
  # Update 'needs' and VERDOX_SUITES to match your jobs
  # ============================================
  verdox-report:
    name: Verdox Report
    runs-on: ubuntu-latest
    if: always()
    needs: [backend-test]
    steps:
      - name: Download all artifacts
        uses: actions/download-artifact@v4
        with:
          path: artifacts

      - name: Build Verdox report
        env:
          VERDOX_RUN_ID: \${{ github.event.inputs.verdox_run_id }}
          VERDOX_REPO: \${{ github.repository }}
          VERDOX_BRANCH: \${{ github.event.inputs.branch }}
          VERDOX_COMMIT: \${{ github.event.inputs.commit_hash }}
          # Configure your suites here:
          # - id: slug for the suite
          # - name: display name
          # - type: unit|integration|e2e|lint|build
          # - result: reference the job outcome
          # - results_file: path to parsed test JSON (for go test jobs)
          VERDOX_SUITES: |
            [
              {"id": "backend-test", "name": "Backend Test", "type": "unit", "result": "\${{ needs.backend-test.result }}", "results_file": "artifacts/backend-test-results/verdox-test-results.json"}
            ]
        run: |${REPORT_COLLECTOR_SCRIPT}

      - name: Upload Verdox results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: verdox-results
          path: verdox-results.json
          retention-days: 7

      - name: Callback to Verdox
        if: always() && github.event.inputs.callback_url != ''
        run: |
          curl -s -X POST "\${{ github.event.inputs.callback_url }}" \\
            -H "Content-Type: application/json" \\
            -d @verdox-results.json || true
`,
  },
  {
    name: "go",
    label: "Go Project (Single Job)",
    description: "Simple single-job Go test workflow with per-test results",
    yaml: `${VERDOX_HEADER}
name: "verdox: Go Tests"

on:
  workflow_dispatch:
    inputs:
      verdox_run_id:
        description: 'Verdox test run ID'
        required: true
      branch:
        description: 'Branch to test'
        required: true
      commit_hash:
        description: 'Commit hash to test'
        required: true
      callback_url:
        description: 'Webhook callback URL'
        required: false

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          ref: \${{ github.event.inputs.branch }}
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Run tests
        id: tests
        run: |
          set +e
          go test -json -race -count=1 ./... 2>&1 | tee test-output.json
          TEST_EXIT=$?
          cp test-output.json test-output.log
          echo "exit_code=$TEST_EXIT" >> $GITHUB_OUTPUT
          exit 0
        continue-on-error: true
      - name: Parse test results
        if: always()
        run: |
          python3 << 'VERDOX_PARSER'
          import json, sys
          tests = {}
          try:
              with open("test-output.json", "r") as f:
                  for line in f:
                      line = line.strip()
                      if not line: continue
                      try: ev = json.loads(line)
                      except: continue
                      test = ev.get("Test", "")
                      if not test: continue
                      if test not in tests:
                          tests[test] = {"status": "", "duration_ms": 0, "output": [], "error": ""}
                      t = tests[test]
                      action = ev.get("Action", "")
                      if action == "output": t["output"].append(ev.get("Output", ""))
                      elif action == "pass":
                          t["status"] = "pass"
                          t["duration_ms"] = int(ev.get("Elapsed", 0) * 1000)
                      elif action == "fail":
                          t["status"] = "fail"
                          t["duration_ms"] = int(ev.get("Elapsed", 0) * 1000)
                          t["error"] = "".join(t["output"])[-2000:]
                      elif action == "skip": t["status"] = "skip"
          except Exception: pass
          results = []
          passed = failed = skipped = 0
          total_ms = 0
          for name, t in tests.items():
              if not t["status"]: continue
              r = {"test_name": name, "status": t["status"], "duration_ms": t["duration_ms"]}
              if t["error"]: r["error_message"] = t["error"]
              results.append(r)
              if t["status"] == "pass": passed += 1
              elif t["status"] == "fail": failed += 1
              elif t["status"] == "skip": skipped += 1
              total_ms += t["duration_ms"]
          status = "failed" if failed > 0 else "passed"
          payload = {
              "version": 2, "status": status,
              "summary": {"total": len(results), "passed": passed, "failed": failed, "skipped": skipped, "duration_ms": total_ms},
              "results": results
          }
          with open("verdox-results.json", "w") as f:
              json.dump(payload, f, indent=2)
          print(f"Verdox: {len(results)} tests ({passed} passed, {failed} failed, {skipped} skipped)")
          VERDOX_PARSER
      - name: Upload results artifact
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: verdox-results
          path: |
            verdox-results.json
            test-output.log
          retention-days: 7
      - name: Callback to Verdox
        if: always() && github.event.inputs.callback_url != ''
        run: |
          curl -s -X POST "\${{ github.event.inputs.callback_url }}" \\
            -H "Content-Type: application/json" \\
            -d @verdox-results.json || true
`,
  },
  {
    name: "node",
    label: "Node.js Project (Single Job)",
    description: "Single-job Node.js test workflow",
    yaml: `${VERDOX_HEADER}
name: "verdox: Node Tests"

on:
  workflow_dispatch:
    inputs:
      verdox_run_id:
        description: 'Verdox test run ID'
        required: true
      branch:
        description: 'Branch to test'
        required: true
      commit_hash:
        description: 'Commit hash to test'
        required: true
      callback_url:
        description: 'Webhook callback URL'
        required: false

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          ref: \${{ github.event.inputs.branch }}
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
      - run: npm ci
      - name: Run tests
        run: npm test -- --ci || true
      - name: Generate Verdox results
        if: always()
        run: echo '{"version":2,"status":"completed","results":[]}' > verdox-results.json
      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: verdox-results
          path: verdox-results.json
          retention-days: 7
      - name: Callback to Verdox
        if: always() && github.event.inputs.callback_url != ''
        run: |
          curl -s -X POST "\${{ github.event.inputs.callback_url }}" \\
            -H "Content-Type: application/json" \\
            -d @verdox-results.json || true
`,
  },
];

export function getTemplate(name: string): WorkflowTemplate | undefined {
  return workflowTemplates.find((t) => t.name === name);
}

export function getDefaultTemplate(): WorkflowTemplate {
  return workflowTemplates[0]; // Multi-Job CI is default
}
