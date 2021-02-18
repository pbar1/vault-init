export BIN             := $(shell basename $(PWD))
export VERSION         := $(shell git describe --tags --always --dirty)
export DOCKER_BUILDKIT := 1

SOURCE      := httos://github.com/pbar1/$(BIN)
IMAGE_REPO  := ghcr.io/pbar1
IMAGE_NAME  := $(IMAGE_REPO)/$(BIN)
BUILD_IMAGE := golang:1

# builds a binary suitable for development
build: build-native

# builds binaries suitable for release
build-release: build-docker

# builds a binary for the current machine only
build-native: clean
	NATIVEONLY=true bash scripts/build.sh

# builds binaries for the app within a consistent, containerized environment
build-docker: clean
	docker run                   \
		--rm --interactive --tty   \
		--workdir="/src"           \
		--volume="$(PWD):/src"     \
		--env="BIN=$(BIN)"         \
		--env="VERSION=$(VERSION)" \
		$(BUILD_IMAGE)             \
		bash scripts/build.sh

# builds a container image for the app
image: build-release
	cat build/Dockerfile.release.in | sed "s|BIN|$(BIN)|g" > build/Dockerfile.release.out
	docker build .                                      \
	--file=build/Dockerfile.release.out                 \
	--build-arg="BIN=$(BIN)"                            \
	--label="org.opencontainers.image.source=$(SOURCE)" \
	--tag=$(IMAGE_NAME):$(VERSION)                      \
	--tag=$(IMAGE_NAME):latest

# publishes a container image version to the registry
image-push: image
	docker push $(IMAGE_NAME):$(VERSION)
	docker push $(IMAGE_NAME):latest

# prints the current version based on git tags
version:
	@echo $(VERSION)

# removes build artifacts from the repository
clean:
	rm -rf bin
