# Prompt: Refine and Architect These Verdox Session 03 Requirements

Review the following feedback and turn it into a concrete product and technical plan for Verdox. Do not drop any statement, intent, or nuance. If a point is phrased as a preference or a rough direction, preserve that and refine it instead of replacing it with a different idea.

Use these reference files while evaluating the requirements:

- `ai/res/docs/schema.json`
- `ai/res/docs/sample_payload.json`

The schema and sample payload represent the type of hierarchy and level of detail I want the test dashboard to support.

In your response:

- preserve every requirement below
- refine the notes into a practical implementation plan
- explain the architecture needed for authoring workflows, ingesting results, storing structured test data, and rendering the dashboard
- call out frontend, backend, workflow-definition, GitHub Actions, payload-generation, and dashboard-model implications
- identify tradeoffs, missing prerequisites, constraints, and recommended implementation order
- clearly state the expected UX and system behavior after the changes

## 1. Replace the Current YAML Builder in Test Suite Creation

Refined points to address:

- While creating a test suite, I want to remove the current YAML builder form for GitHub Actions because it is not flexible enough.
- Instead of the YAML builder form, I want a complete editor to be available so the workflow can be edited directly.
- That editor should open with a pre-set template rather than an empty file.
- The pre-set template should include things like workflow name, output stage, and other required Verdox workflow structure so users can edit the full workflow entirely instead of being limited by a narrow form.
- Refine what the right authoring experience should be here: full editor only, full editor plus helper fields, or full editor backed by a template system that still preserves complete edit freedom.
- Define the expected behavior clearly: test suite creation should remain guided enough to produce valid Verdox-compatible workflows, but flexible enough for advanced GitHub Actions use cases that the current builder cannot support.

## 2. Extend the Test Suite Dashboard Into a Detailed Hierarchical Test Dashboard

Reference requirements:

- See `ai/res/docs/schema.json` for the type of schema I want the dashboard to incorporate.
- I have taken the example of the Consul repo to build the sample payload in `ai/res/docs/sample_payload.json`.

Refined points to address:

- I want to extend the test suite dashboard into a more detailed test dashboard that can represent the hierarchy and detail level shown in the schema and sample payload.
- Help me architect such a system for a detailed test system, not just a flat test-suite list.
- The hierarchy should be explicit and first-class in the product model: `test suites -> tests -> test cases`.
- The dashboard should be able to represent repository-level metadata, run-level metadata, summary rollups, suites, tests, and individual cases in a structured hierarchy.
- Verdox should treat test suites as the primary top-level grouping, each suite should contain tests, and each test should contain test cases.
- The dashboard UX should make it easy to move through this hierarchy without losing context, especially for large repositories and large runs.
- I want a details page for every suite so users can open a suite and inspect its tests, cases, failures, timings, logs, and related metadata in a focused view.
- Refine what should appear on the suite details page, how deep navigation should work from suite to test to test case, and what information should remain visible at each level.
- The architecture should account for the type of fields present in the schema, such as repo, run ID, branch, commit SHA, timestamp, summary metrics, suite-level stats, test-level stats, case-level status, durations, error details, stack traces, retry counts, and log links.
- Refine how Verdox should model, store, validate, version, and render this hierarchy so the dashboard remains usable even when the payload becomes large and deeply nested.
- Clarify what should be computed by Verdox, what should be supplied by the workflow payload, and what should be derived during ingestion.
- Define the expected behavior clearly: Verdox should render a detailed, navigable, hierarchy-aware dashboard for complex repositories similar to Consul, with clear traversal from test suites to tests to test cases, and with a dedicated details page for each suite instead of only showing simplified suite-level results.

## 3. Support Repositories That Already Have Existing CI Test Infrastructure

Refined points to address:

- Some repositories already contain CI tests, such as Consul.
- I want a way to configure generation of a dashboard payload for Verdox to render, even when the repository already has an existing test infrastructure.
- Verdox should be able to scan and build suites from existing test infrastructure into the dashboard instead of requiring everything to be rewritten from scratch in Verdox-specific workflows.
- That scan can be done with AI if useful.
- Manual options should also be available instead of relying only on AI.
- Refine how Verdox should support both approaches:
  - AI-assisted scanning or inference of existing CI/test structure
  - manual configuration and mapping of suites, tests, outputs, and artifacts
- Clarify how Verdox should detect existing workflows, test commands, artifacts, logs, package structures, and output formats, and then map them into the dashboard schema.
- Clarify whether Verdox should support adapters, parsers, or transform layers that convert existing CI outputs into the schema represented by `ai/res/docs/schema.json`.
- Define the expected behavior clearly: for repositories with mature existing CI, Verdox should help onboard them into the dashboard model through scanning, mapping, and configuration rather than forcing a full greenfield workflow rewrite.

## 4. Response Expectation

Do not simplify these notes by removing details. I want them rewritten and addressed as a well-structured implementation prompt that keeps my original meaning intact while making the requirements clearer, more detailed, and easier to act on.
