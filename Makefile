test:
ifdef run
	@go test -v -tags=delve ./... -run=$(run)
else
	@go test -tags=delve ./...
endif

debug:
	@dlv debug cmd/rms_server.go --build-flags="-tags=delve"

debug_test:
ifdef run
	@dlv test . --build-flags="-tags=delve" -- -test.run $(run)
else
	@dlv test . --build-flags="-tags=delve"
endif

coverage:
	@go test -tags=delve -coverprofile=coverage
	@go tool cover -func=coverage

coverage_html: coverage
	@go tool cover -html=coverage

clean:
	rm coverage
