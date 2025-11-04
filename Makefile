.PHONY: run wire build deploy
#run
run:
	go run ./cmd
wire:
	wire ./cmd
# build
build: wire
	rm -rf ./bin
	mkdir -p bin/ && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ./bin/Gwatch cmd/main.go cmd/wire_gen.go
	upx -9 ./bin/Gwatch
	echo "upx 压缩完成"
	@echo "编译完成: bin/Gwatch"
	@ls -lh bin/Gwatch

deploy169:
	scp bin/Gwatch config/config_server.yml youxihu@172.235.216.169:/tmp

deploy171:
	scp bin/Gwatch config/config_client.yml youxihu@172.235.216.171:/tmp

deploy172:
	scp bin/Gwatch config/config_client.yml youxihu@172.235.216.172:/tmp