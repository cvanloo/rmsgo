test:
	@go test ./...

debug:
	@dlv debug cmd/rms_server.go --build-flags="-tags=delve"

