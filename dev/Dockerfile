FROM golang:alpine
LABEL maintainer="Dmitry Rodin <madiedinro@gmail.com>"

RUN apk add git --no-cache

ENV HOST=0.0.0.0
ENV PORT=8080
ARG RELEASE=master

EXPOSE ${PORT}
RUN go get github.com/cespare/reflex

WORKDIR /go/src/heavyload

CMD ["/go/bin/reflex", "--start-service", "-g", "*.go", "--", "sh", "-c", "go run main.go"]


