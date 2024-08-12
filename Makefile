build:
	rm -rf ./.out/*
	go build -o ./.out/app

buildr:
	go build -race -o ./out/app

run: build
	chmod +xw ./.out/app
	./out/app

run_env: build
	chmod +xw ./.out/app
	ENV_PATH=.example.env ./.out/app

test:
	go test */***
