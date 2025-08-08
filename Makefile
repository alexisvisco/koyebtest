.PHONY: test

test:
	go test ./...

mocks:
    mockery --with-expecter --dir mocks --filename "{{.InterfaceNameSnake}}.go" --structname "{{.InterfaceName}}" --disable-version-string
