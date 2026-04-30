SHELL := /bin/bash

# Avoid leaking a host or CI-provided GOROOT into Go commands.
GO := env -u GOROOT go
TERRAFORM ?= terraform

.PHONY: help lint test build check format lambda-artifacts terraform-fmt terraform-lint terraform-validate terraform-test terraform-policy terraform-check workflow-lint ci
.PHONY: pnpm-install pnpm-lint pnpm-typecheck pnpm-test pnpm-build go-fmt go-vet go-test go-build go-lint
.PHONY: ci-ts ci-go ci-terraform

help:
	@printf '%s\n' \
		'Targets:' \
		'  make lint              Run formatting checks and language linters' \
		'  make test              Run unit tests' \
		'  make build             Run build checks' \
		'  make check             Run local development checks' \
		'  make ci                Run the pull request gate locally' \
		'  make lambda-artifacts  Build dev Lambda zip artifacts' \
		'  make format            Format supported files' \
		'  make terraform-test    Run native Terraform module and root tests' \
		'  make terraform-policy  Run Terraform security policy checks' \
		'  make terraform-check   Run Terraform fmt, validate, test, lint, and policy'

pnpm-install:
	pnpm install --frozen-lockfile

pnpm-lint:
	pnpm run lint

pnpm-typecheck:
	pnpm run typecheck

pnpm-test:
	pnpm test

pnpm-build:
	pnpm run build

go-test:
	cd services && $(GO) test ./...

go-build:
	cd services && $(GO) build ./...

go-fmt:
	@bash scripts/ci/go-fmt-check.sh

go-vet:
	cd services && $(GO) vet ./...

go-lint:
	$(MAKE) go-fmt
	$(MAKE) go-vet

lint: pnpm-lint go-lint terraform-check workflow-lint

test: pnpm-test go-test

build: pnpm-build go-build

check: ci

lambda-artifacts:
	bash scripts/build/lambda-artifacts.sh

ci-ts:
	$(MAKE) pnpm-lint
	$(MAKE) pnpm-typecheck
	$(MAKE) pnpm-test
	$(MAKE) pnpm-build

ci-go:
	$(MAKE) go-lint
	$(MAKE) go-test
	$(MAKE) go-build

ci-terraform:
	$(MAKE) terraform-check

workflow-lint:
	$(GO) run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7

ci: ci-ts ci-go ci-terraform workflow-lint

format:
	pnpm run format
	env -u GOROOT gofmt -w services
	$(TERRAFORM) fmt -recursive infra/terraform

terraform-fmt:
	$(TERRAFORM) fmt -recursive -check infra/terraform

terraform-lint:
	bash scripts/ci/terraform-lint.sh

terraform-policy:
	bash scripts/ci/terraform-policy.sh

terraform-validate:
	TERRAFORM="$(TERRAFORM)" bash scripts/ci/terraform-validate.sh

terraform-test:
	$(TERRAFORM) -chdir=infra/terraform/envs/dev init -backend=false -input=false
	$(TERRAFORM) -chdir=infra/terraform/envs/dev test
	$(TERRAFORM) -chdir=infra/terraform/envs/stg init -backend=false -input=false
	$(TERRAFORM) -chdir=infra/terraform/envs/stg test
	$(TERRAFORM) -chdir=infra/terraform/envs/prod init -backend=false -input=false
	$(TERRAFORM) -chdir=infra/terraform/envs/prod test
	$(TERRAFORM) -chdir=infra/terraform/modules/api-http init -backend=false -input=false
	$(TERRAFORM) -chdir=infra/terraform/modules/api-http test
	$(TERRAFORM) -chdir=infra/terraform/modules/compute-lambda init -backend=false -input=false
	$(TERRAFORM) -chdir=infra/terraform/modules/compute-lambda test
	$(TERRAFORM) -chdir=infra/terraform/modules/data-dynamodb init -backend=false -input=false
	$(TERRAFORM) -chdir=infra/terraform/modules/data-dynamodb test
	$(TERRAFORM) -chdir=infra/terraform/modules/eventing init -backend=false -input=false
	$(TERRAFORM) -chdir=infra/terraform/modules/eventing test
	$(TERRAFORM) -chdir=infra/terraform/modules/observability init -backend=false -input=false
	$(TERRAFORM) -chdir=infra/terraform/modules/observability test
	$(TERRAFORM) -chdir=infra/terraform/modules/secrets init -backend=false -input=false
	$(TERRAFORM) -chdir=infra/terraform/modules/secrets test
	$(TERRAFORM) -chdir=infra/terraform/modules/storage-s3 init -backend=false -input=false
	$(TERRAFORM) -chdir=infra/terraform/modules/storage-s3 test

terraform-check:
	$(MAKE) terraform-fmt
	$(MAKE) terraform-validate
	$(MAKE) terraform-test
	$(MAKE) terraform-lint
	$(MAKE) terraform-policy
