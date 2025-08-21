.PHONY: gazelle build test run compose-up compose-down

gazelle:
	bazel run //:gazelle

build:
	bazel build //...

test:
	bazel test //...

run-api:
	bazel run //cmd/api:api

compose-up:
	docker compose -f deploy/docker-compose.yml up -d

compose-down:
	docker compose -f deploy/docker-compose.yml down