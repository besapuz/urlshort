test: 
	@echo "Запуск тестов"
	@go test -v -cover ./...

.PHONY: proto-gen
proto-gen:
	@echo "🔄 Generating proto files..."
	@rm -rf github.com/besapuz/urlshort/github.com
	@rm -f api/proto/*.pb.go
	@protoc \
		--proto_path=. \
		--go_out=. \
		--go_opt=module=github.com/besapuz/urlshort \
		--go-grpc_out=. \
		--go-grpc_opt=module=github.com/besapuz/urlshort \
		api/proto/shortener.proto
	@echo "✅ Proto files generated successfully!"

.PHONY: proto-clean
proto-clean:
	@echo "🧹 Cleaning proto generated files..."
	@rm -rf github.com/besapuz/urlshort/github.com
	@rm -f api/proto/*.pb.go
	@echo "✅ Clean completed!"

.PHONY: proto-regen
proto-regen: proto-clean proto-gen
	@echo "🔄 Proto files regenerated!"