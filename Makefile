export BIN             := $(shell basename $(PWD))
export VERSION         := $(shell git describe --tags --always --dirty)
export DOCKER_BUILDKIT := 1

SOURCE      := httos://github.com/pbar1/$(BIN)
IMAGE_REPO  := ghcr.io/pbar1
IMAGE_NAME  := $(IMAGE_REPO)/$(BIN)
BUILD_IMAGE := golang:1

build: build-docker

build-native: clean
	bash scripts/build.sh

build-docker: clean
	docker run                   \
		--rm --interactive --tty   \
		--workdir="/src"           \
		--volume="$(PWD):/src"     \
		--env="BIN=$(BIN)"         \
		--env="VERSION=$(VERSION)" \
		$(BUILD_IMAGE)             \
		bash scripts/build.sh

image: build
	cat build/Dockerfile.release.in | sed "s|BIN|$(BIN)|g" > build/Dockerfile.release.out
	docker build .                                      \
	--file=build/Dockerfile.release.out                 \
	--build-arg="BIN=$(BIN)"                            \
	--label="org.opencontainers.image.source=$(SOURCE)" \
	--tag=$(IMAGE_NAME):$(VERSION)                      \
	--tag=$(IMAGE_NAME):latest

image-push: image
	docker push $(IMAGE_NAME):$(VERSION)
	docker push $(IMAGE_NAME):latest

clean:
	rm -rf bin
