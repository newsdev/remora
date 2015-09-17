all:
	docker build -t remora-build .
	docker run --rm -v $(CURDIR)/bin:/opt/bin remora-build go build -o /opt/bin/remora .

clean:
	docker rmi remora-build || true
	rm -r build
