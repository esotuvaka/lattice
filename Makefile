run: build
	@./bin/lattice

build:
	@go build -o bin/lattice .