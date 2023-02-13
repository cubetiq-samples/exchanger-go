all: build

build:
	go build -o main .

run:
	./main

docker:
	docker build -t currency-exchange-serverless .
