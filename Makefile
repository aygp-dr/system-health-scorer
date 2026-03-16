.PHONY: build run test clean

build:
	go build -o bin/system-health-scorer .

run: build
	./bin/system-health-scorer

test:
	go test ./...

clean:
	rm -rf bin/
