.PHONY: run wire build deploy
#run
run:
	go run ./cmd
wire:
	wire ./cmd
# build
build:
	rm -rf ./bin
	mkdir -p bin/ && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -o ./bin/Gwatch ./cmd
	upx  ./bin/*

deploy169:
	scp bin/Gwatch youxihu@172.235.216.169:/tmp

deploy171:
	scp bin/Gwatch youxihu@172.235.216.171:/tmp

deploy172:
	scp bin/Gwatch youxihu@172.235.216.172:/tmp