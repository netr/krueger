.PHONY: build

garb:
	garble build -o build/krueger
build:
	go build --trimpath -o build/krueger