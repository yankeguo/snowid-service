FROM golang:1.19 AS builder
ENV CGO_ENABLED 0
WORKDIR /go/src/app
ADD . .
RUN go build -o /snowid-service

FROM alpine:3.18
RUN apk add --no-cache tini
COPY --from=builder /snowid-service /snowid-service
ENTRYPOINT ["tini", "--"]
CMD ["/snowid-service"]
