build:
    go build -o bin/moonshine .

test:
    go test ./...

lint:
    golangci-lint run

install: build
    cp bin/moonshine /usr/local/bin/moonshine
    ln -sf /usr/local/bin/moonshine /usr/local/bin/moon
    ln -sf /usr/local/bin/moonshine /usr/local/bin/ms

clean:
    rm -rf bin/