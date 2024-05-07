FROM golang:alpine
LABEL maintainer="Dmitry Rodin <madiedinro@gmail.com>"

ENV HOST=0.0.0.0
ENV PORT=8080
ENV GIN_MODE=release
EXPOSE ${PORT}




#cachebust
ARG RELEASE=master
WORKDIR /go/src/heavyload
COPY . .
RUN mkdir -p upload

RUN go mod init heavyload
RUN go mod tidy
RUN rm -rf vendor
RUN go build . 
RUN du -h
# RUN ls heavyload


# RUN go build
# RUN go install
CMD ["/go/src/heavyload/heavyload"]
