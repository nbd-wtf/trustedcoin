trustedcoin: $(shell find . -name "*.go")
	go build -ldflags='-s -w' -o ./trustedcoin
