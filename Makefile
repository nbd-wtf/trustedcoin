trustedcoin: $(shell find . -name "*.go")
	CC=$$(which musl-gcc) CGO_ENABLED=0 go build -trimpath -ldflags='-d -s -w' -o ./trustedcoin
