FROM golang:alpine

RUN apk add --no-cache git

ENV GOBIN /go/bin

RUN mkdir /app

RUN mkdir /go/src/app

COPY . /go/src/app

WORKDIR /go/src/app

RUN go get -u github.com/golang/dep/...

RUN dep ensure

RUN go build -o /app/*.go .

CMD [ "/app/poller" ]
