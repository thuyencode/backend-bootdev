FROM golang:alpine AS build

WORKDIR /go/src/app

COPY . .
COPY . .
COPY go.mod go.sum ./
RUN go mod download

ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /go/bin/app

FROM gcr.io/distroless/static-debian13

COPY --from=build /go/bin/app /
COPY --from=build /go/src/app/.env /

EXPOSE 8080

CMD ["./app"]