build:
	go build
docker: build
	docker build . -t goserver:latest
run:
	godotenv -f .env go run .
clean:
	go clean