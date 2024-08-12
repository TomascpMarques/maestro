build:
	rm -rf .out/*
	go build -o ./out/app

buildr:
	go build -race -o ./out/app

run: build
	chmod +x ./out/app
	./out/app

test:
	go test */***
