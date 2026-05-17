# Staging Terraform Root

This root is a placeholder. It validates partial S3 backend wiring, provider configuration, and environment naming only; it does not create AWS resources yet.

Native Terraform tests live in `tests/stg_contract.tftest.hcl`. They assert that the default environment is `stg` and non-staging environment names are rejected.

The staging environment should either call a shared environment module or define its own resources before it is used for deployment.
