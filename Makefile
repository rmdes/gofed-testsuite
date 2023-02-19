-include environ.inc
.PHONY: dev build install image clean

GOCMD=go
IMAGE := r.mills.io/prologic/testsuite
TAG := latest

all: build

dev : DEBUG=1
dev : build
	@./testsuite

build:
	@$(GOCMD) build

ifeq ($(PUBLISH), 1)
image:
	@docker buildx build --platform linux/amd64,linux/arm64 --push -t $(IMAGE):$(TAG) .
else
image:
	@docker build  -t $(IMAGE):$(TAG) .
endif

clean:
	@git clean -f -d -X
