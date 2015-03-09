FROM golang:1.4.2-onbuild
ENTRYPOINT ["go-wrapper", "run"]
