FROM golang

WORKDIR /go/src/app

COPY index.html index.html

RUN go get -u github.com/shurcooL/goexec

EXPOSE 8080

CMD [ "goexec", "http.ListenAndServe(`:8080`, http.FileServer(http.Dir(`.`)))" ]