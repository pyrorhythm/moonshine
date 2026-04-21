build:
    go build -o bin/ms ./cmd/ms

test:
    go test ./...

lint:
    golangci-lint run

install: build
    cp bin/ms /usr/local/bin/ms
    ln -sf /usr/local/bin/ms /usr/local/bin/moon
    ln -sf /usr/local/bin/ms /usr/local/bin/moonshine

clean:
    rm -rf bin/

snapshot:
    goreleaser release --snapshot --clean

updsum SEMVER:
	sleep 3
	curl https://sum.golang.org/lookup/github.com/pyrorhythm/fn@{{SEMVER}}

tag-push SEMVER:
	git tag {{SEMVER}}
	git push origin {{SEMVER}}

commit-push SEMVER:
    git add . ; git commit -m "release: {{SEMVER}}"
    git tag {{SEMVER}}
    git push ; git push origin {{SEMVER}}

release SEMVER: test (commit-push SEMVER) (updsum SEMVER)
