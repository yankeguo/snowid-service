FROM golang:1.19 AS builder
ENV CGO_ENABLED 0
WORKDIR /go/src/app
ADD . .
RUN go build -o /snowid-service

FROM scratch
COPY --from=builder /snowid-service /snowid-service
CMD ["/snowid-service"]
