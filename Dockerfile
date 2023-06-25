FROM rust:alpine as build
WORKDIR /usr/src/vault-init
COPY . .
RUN cargo install --path .

FROM alpine
COPY --from=build /usr/local/cargo/bin/vault-init /usr/local/bin/vault-init
CMD ["vault-init"]
