build:
    go build -o bin/ms ./cmd/ms

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

snapshot:
    goreleaser release --snapshot --clean

updsum SEMVER:
	sleep 3
	curl https://sum.golang.org/lookup/pyrorhythm.dev/moonshine@{{SEMVER}}

tag-push SEMVER:
	git tag {{SEMVER}}
	git push origin {{SEMVER}}

release SEMVER: test (tag-push SEMVER) (updsum SEMVER)
