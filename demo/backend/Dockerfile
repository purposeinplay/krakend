FROM golang

WORKDIR /go/src/app

COPY main.go main.go

RUN go build

EXPOSE 8080

CMD ["./app"]