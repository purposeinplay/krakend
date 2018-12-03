FROM golang

WORKDIR /go/src/app

COPY main.go main.go

RUN go get gopkg.in/olahol/melody.v1
RUN go build

EXPOSE 8080

CMD ["./app"]