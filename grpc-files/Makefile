generate-proto:
	make build-protoc-image
	./generate.sh
build-protoc-image:
	docker build -f Dockerfile.protoc -t protoc:latest .