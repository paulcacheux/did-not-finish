FROM golang:1.19-bullseye AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /did-not-finish .

FROM amazonlinux:2022

COPY --from=builder /did-not-finish .

CMD "./did-not-finish"