FROM golang:alpine

RUN apk add --no-cache git

ENV GOBIN /go/bin

RUN mkdir /app

RUN mkdir /go/src/app

COPY . /go/src/app

WORKDIR /go/src/app

RUN apk add gcc libc-dev
RUN apk add --repository http://dl-cdn.alpinelinux.org/alpine/edge/testing geos geos-dev
RUN go get github.com/Spatially/gogeos/geos

RUN go build -o /app/*.go .

CMD [ "/app/poller" ]
