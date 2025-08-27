.PHONY: run build
#run
run:
	go run cmd/main.go

# build
build:
	rm -rf ./bin
	mkdir -p bin/ && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -o ./bin/Gwatch ./cmd/main.go
	upx  ./bin/*

deploy:
	scp bin/Gwatch youxihu@172.235.216.169:/tmp