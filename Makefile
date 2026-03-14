build:
	go build -o swarmsim .

windows:
	GOOS=windows GOARCH=amd64 go build -o swarmsim.exe .

linux:
	GOOS=linux GOARCH=amd64 go build -o swarmsim-linux .

wasm:
	mkdir -p docs
	GOOS=js GOARCH=wasm go build -o docs/swarmsim.wasm .
	cp $$(go env GOROOT)/lib/wasm/wasm_exec.js docs/

test:
	go test ./... -v -count=1

cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	go vet ./...

clean:
	rm -f swarmsim swarmsim.exe swarmsim-linux swarmsim.wasm wasm_exec.js coverage.out coverage.html

.PHONY: build windows linux wasm test cover lint clean
