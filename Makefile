SERVICE=searchd
GOFLAGS?=-trimpath
.PHONY: test vet race run build smoke clean
test:
	go test ./...
vet:
	go vet ./...
race:
	go test -race ./...
build:
	go build $(GOFLAGS) -o build/$(SERVICE) ./cmd/searchd
run:
	go run ./cmd/searchd
smoke:
	./scripts/smoke.sh
clean:
	rm -rf build/searchd data/index.wal
