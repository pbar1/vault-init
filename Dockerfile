FROM alpine:latest AS build
ARG TARGETARCH
WORKDIR /
COPY bin .
RUN mv vault-init_linux_$TARGETARCH vault-init

FROM alpine:latest
COPY --from=build /vault-init /
CMD ["/vault-init"]
