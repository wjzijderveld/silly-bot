FROM golang:1.22

WORKDIR /usr/src/app

COPY *.go go.* .
RUN go build -o silly .

CMD ["./silly"]
