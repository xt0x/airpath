# Staging Terraform Root

This root is a placeholder. It validates partial S3 backend wiring, provider configuration, and environment naming only; it does not create AWS resources yet.

Contract tests live in `tests/stg_contract.tftest.hcl` and `../stg-prod-contract.test.ts`. They assert that the default environment is `stg`, non-staging environment names are rejected, the shared S3 backend is declared for init-time backend config, and no deployable Terraform resources, data sources, or modules are declared.

The staging environment should either call a shared environment module or define its own resources before it is used for deployment.
