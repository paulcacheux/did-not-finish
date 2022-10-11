.PHONY: test test-docker

test:
	go run . -repos ./fixtures/repos -vars ./fixtures/vars

test-docker:
	docker build . -t dnf
	docker run -it dnf