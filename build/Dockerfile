FROM alpine
RUN apk add --update openssl && rm -rf /var/cache/apk/*

# Install the remora binary
ENV REMORA_VERSION=0.0.1
RUN \
  cd /usr/local/bin && \
  wget https://github.com/newsdev/remora/releases/download/v$REMORA_VERSION/remora && \
  chmod +x remora

ENTRYPOINT ["remora"]
