FROM alpine:latest
LABEL org.opencontainers.image.source=https://github.com/pbar1/vault-init

COPY bin/vault-init_linux_amd64 /usr/local/bin/vault-init

RUN chmod +x /usr/local/bin/vault-init

CMD ["/usr/local/bin/vault-init"]
