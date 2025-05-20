# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=elasticphp-exporter
VERSION?=1.0.0
BUILD_DIR=build
DOCKER_REPO=elasticphphq
IMAGE_NAME=php:8.4-fpm-bookworm
CONTAINER_NAME=elasticphp-dev

build:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v .
	CGO_ENABLED=0 $(GOBUILD) -o $(BUILD_DIR)/arm64-$(BINARY_NAME) -v .

build-docker:
	docker build -t $(IMAGE_NAME) .

build-pbf:
	clang -O2 -g -target bpf -c ./bpf/monitor.c \
		-D__TARGET_ARCH_arm64 \
		-o ./internal/ebpf/monitor.o \
		-I/usr/include -I/usr/src/linux-headers-$(uname -r)/include \
		-I/usr/src/linux-headers-$(shell uname -r)/include \
		-D __BPF_TRACING__

run:
	docker run -it --rm \
		--name $(CONTAINER_NAME) \
		-v $(CURDIR):/app \
		-w /app \
		$(IMAGE_NAME) bash

shell:
	docker exec -it $(CONTAINER_NAME) bash

stop:
	docker stop $(CONTAINER_NAME) || true

clean:
	docker rmi $(IMAGE_NAME) || true