build:
	rm -rf ./.out/*
	rm -rf ./rng
	go build -o ./.out/app

buildr:
	go build -race -o ./out/app

run: build
	chmod +xw ./.out/app
	./.out/app

run_env: build
	chmod +xw ./.out/app
	ENV_PATH=.example.env ./.out/app

testA:
	go test -v ./...

migrate:
	migrate create -ext .sqlite -dir migrations -format unix -tz "UTC" $(MIGN)