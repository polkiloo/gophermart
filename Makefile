coverage:
	@set -euo pipefail; \
	pkgs="$$(go list ./... | paste -sd, -)"; \
	go test -cover -covermode=atomic -race -coverpkg="$$pkgs" \
	  -coverprofile=coverage.out ./... >/dev/null; \
	go tool cover -func=coverage.out | tail -n1

coverage-html: coverage
	@go tool cover -html=coverage.out -o coverage.html && \
	echo "Report: coverage.html"