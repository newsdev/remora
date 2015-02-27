all: bin/remora

release: bin/remora
	aws s3 cp bin/remora s3://newsdev-pub/bin/remora

bin/remora: bin
	docker build -t remora-build $(CURDIR) && docker run --rm -v $(CURDIR)/bin:/opt/bin remora-build cp /go/bin/app /opt/bin/remora

bin:
	mkdir -p bin

clean:
	rm -rf bin
	docker rmi remora-build || true
