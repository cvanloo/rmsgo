.PHONY: test debug debug_test coverage coverage_html clean help

test:
ifdef run
	@go test -v -tags=delve ./... -run=$(run)
else
	@go test -tags=delve ./...
endif

debug_test:
ifdef run
	@dlv test . --build-flags="-tags=delve" -- -test.run $(run)
else
	@dlv test . --build-flags="-tags=delve"
endif

debug:
	@dlv debug cmd/rms_server.go --build-flags="-tags=delve"

coverage: coverage.out
	@go tool cover -func=coverage.out

coverage_html: coverage.out
	@go tool cover -html=coverage.out

clean:
	-rm coverage.out

coverage.out: .FORCE
	@go test -tags=delve -coverprofile=coverage.out ./...

help:
	@cat Makefile | grep -E "^\w+:$:"

.FORCE:
