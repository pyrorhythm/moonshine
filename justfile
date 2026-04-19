build:
    goutil build -o build/moonshine .

test:
    goutil test ./...

lint:
    golangci-lint run

install: build
    cp build/moonshine /usr/local/bin/moonshine
    ln -sf /usr/local/bin/moonshine /usr/local/bin/moon
    ln -sf /usr/local/bin/moonshine /usr/local/bin/ms

clean:
    rm -rf bin/