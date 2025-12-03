# Makefile senzill només per Docker

.PHONY: build run shell clean

build:
	docker build -t devcon .

run:
	docker run --rm -it -v $(PWD):/app --name devcon devcon

shell:
	docker exec -it devcon zsh

clean:
	docker rm -f devcon || true
