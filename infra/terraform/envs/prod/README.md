# Production Terraform Root

This root is a placeholder. It validates partial S3 backend wiring, provider configuration, and environment naming only; it does not create AWS resources yet.

Native Terraform tests live in `tests/prod_contract.tftest.hcl`. They assert that the default environment is `prod` and non-production environment names are rejected.

The production environment should either call a shared environment module or define its own resources before it is used for deployment.
