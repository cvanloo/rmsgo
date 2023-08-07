test:
	@go test -tags=delve ./...

debug:
	@dlv debug cmd/rms_server.go --build-flags="-tags=delve"

