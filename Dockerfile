FROM golang:1.24-alpine AS build
WORKDIR /
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY *.go ./
RUN go build -o /main .

FROM alpine:latest
WORKDIR /root/
COPY --from=build /main .
EXPOSE 8080
CMD ["./main"]