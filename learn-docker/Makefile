build:
	GOOS=linux GOARCH=amd64 go build
image: build
	docker build . -t thuyencode/goserver:$(VERSION)
push-image: image
	docker push thuyencode/goserver:$(VERSION)
run:
	godotenv -f .env go run .
clean:
	go clean