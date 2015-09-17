FROM golang:1.5.1
ENV CGO_ENABLED=0
ADD . /go/src/github.com/newsdev/remora
WORKDIR /go/src/github.com/newsdev/remora
RUN \
  go get github.com/tools/godep && \
  godep restore
