build:
	@echo "Building..."
	@GOOS=js GOARCH=wasm go build -o ./static/engine.wasm .

serve:
	@echo "Serving http://localhost:8080..."
	@go run cmd/server.go