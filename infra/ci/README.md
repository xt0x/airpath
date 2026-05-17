# CI

This directory contains tests for repository-level CI and developer-tooling contracts.

Run the CI contract tests with:

```bash
pnpm exec vitest run infra/ci
```

Run the same Gitleaks command used by the local pull request gate with:

```bash
make gitleaks
```

The GitHub Actions workflow runs Gitleaks against full Git history and redacts secret values from output.

The root `.gitleaks.toml` keeps the default Gitleaks rules enabled and only narrows confirmed false positives.

The developer script contract keeps root workspace commands, such as `pnpm dev`,
delegating to the package that owns the implementation.
