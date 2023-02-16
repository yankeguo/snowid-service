FROM golang:1.19 AS builder
ENV CGO_ENABLED 0
WORKDIR /go/src/app
ADD . .
RUN go build -mod vendor -o /snowid-service

FROM alpine:3.17
RUN apk add --no-cache tini
COPY --from=builder /snowid-service /snowid-service
ENTRYPOINT ["tini", "--"]
CMD ["/snowid-service"]
