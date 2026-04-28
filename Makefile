SHELL := /bin/bash

GO_ENV := env -u GOROOT
TFLINT ?= tflint

.PHONY: help lint test build format terraform-fmt terraform-lint terraform-validate terraform-check workflow-lint ci
.PHONY: pnpm-install pnpm-lint pnpm-test pnpm-build go-fmt go-vet go-test go-lint
.PHONY: pnpm-typecheck frontend-build ci-ts ci-go ci-terraform

help:
	@printf '%s\n' \
		'Targets:' \
		'  make lint               Run formatting checks and language linters' \
		'  make test               Run unit tests' \
		'  make build              Run build checks' \
		'  make format             Format supported files' \
		'  make pnpm-install       Install pnpm dependencies with the lockfile' \
		'  make terraform-fmt      Check Terraform formatting' \
		'  make terraform-lint     Run TFLint across Terraform roots' \
		'  make terraform-validate Run Terraform validate across roots' \
		'  make terraform-check    Run Terraform fmt, validate, and lint' \
		'  make workflow-lint      Run GitHub Actions workflow lint' \
		'  make ci                 Run the pull request gate locally'

pnpm-install:
	pnpm install --frozen-lockfile

pnpm-lint:
	pnpm run lint

pnpm-test:
	pnpm test

pnpm-build:
	pnpm run build

pnpm-typecheck:
	pnpm run typecheck

frontend-build:
	pnpm --filter @airpath/web build

go-test:
	cd services && $(GO_ENV) go test ./...

go-fmt:
	@bash scripts/ci/go-fmt-check.sh

go-vet:
	cd services && $(GO_ENV) go vet ./...

go-lint: go-fmt go-vet

lint: pnpm-lint go-lint terraform-check

test: pnpm-test go-test

build: pnpm-build go-test

ci-ts:
	pnpm run lint
	pnpm run typecheck
	pnpm test
	pnpm run build

ci-go: go-lint go-test

ci-terraform: terraform-check

workflow-lint:
	$(GO_ENV) go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7

ci: ci-ts ci-go ci-terraform workflow-lint

format:
	pnpm run format
	$(GO_ENV) gofmt -w services
	terraform fmt -recursive infra/terraform

terraform-fmt:
	terraform fmt -recursive -check infra/terraform

terraform-lint:
	@TFLINT="$(TFLINT)" TFLINT_CONFIG="$(CURDIR)/.tflint.hcl" bash scripts/ci/terraform-lint.sh

terraform-validate:
	bash scripts/ci/terraform-validate.sh

terraform-check: terraform-fmt terraform-validate terraform-lint
