build:
	go build -o ./bin/blockchain_1_0

run: build
	./bin/blockchain_1_0

test:
	go test -v ./...