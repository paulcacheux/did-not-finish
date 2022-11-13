.PHONY: test test-docker

test-al2022:
	go run . -repos ./fixtures/al2022/repos -vars ./fixtures/al2022/vars -release-ver 2022.0.20220928

test-fedora:
	go run . -repos ./fixtures/fedora/repos -vars ./fixtures/fedora/vars -release-ver 36

test-docker:
	docker build . -t dnf
	docker run -it dnf