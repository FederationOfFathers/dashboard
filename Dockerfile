FROM golang:1.18-alpine AS build
ENV CGO_ENABLED=0
RUN apk update && apk add git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod vendor
COPY . ./
RUN go build
ENTRYPOINT ["/app/dashboard"]

FROM alpine
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=build /app/dashboard .
CMD ["./dashboard"]
