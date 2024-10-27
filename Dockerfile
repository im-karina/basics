FROM golang:1.23

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download && go mod verify

COPY . .
ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target="/root/.cache/go-build" go build -v -o /usr/local/bin/app main.go

CMD ["app", "db:migrate", "serve"]
