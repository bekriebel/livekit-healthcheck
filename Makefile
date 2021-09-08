.PHONY: alpine build
alpine:
	docker run --rm -it -v "${PWD}":/usr/src/app -w /usr/src/app golang:1.17-alpine go build -v

build:
	docker run --rm -v "${PWD}":/usr/src/app -w /usr/src/app golang:1.17 go build -v