# CI Design

The CI area owns repository-level automation and developer-tooling contracts that do not belong to a single runtime package. Tests in this directory verify the expected GitHub Actions wiring and root workspace scripts so security, workflow checks, and common local commands remain part of the pull request gate.

## Responsibilities

- Keep the top-level CI workflow connected to reusable workflow files under `.github/workflows`.
- Verify that the secret scanning workflow checks full Git history with redacted output.
- Verify that root package scripts delegate package-specific developer commands to the owning workspace package.
- Avoid embedding credentials in workflow files. Runtime secrets must come from GitHub-provided tokens, GitHub repository secrets, or environment variables.

## Secret Scanning

Gitleaks runs in `ci-security.yml` through the official `ghcr.io/gitleaks/gitleaks` container image. The workflow checks out full Git history with `fetch-depth: 0` because secrets may appear in previous commits even when the current tree is clean. The scanner uses `--redact` so detected secret values are not printed in CI logs.

The root `.gitleaks.toml` extends the built-in default rules. Local allowlists must be narrow, documented, and limited to confirmed false positives.
