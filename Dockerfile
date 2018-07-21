FROM golang:alpine

RUN apk add --no-cache git

ENV GOBIN /go/bin

RUN mkdir /app

RUN mkdir /go/src/app

COPY . /go/src/app

WORKDIR /go/src/app

RUN apk add gcc libc-dev
RUN apk add --repository http://dl-cdn.alpinelinux.org/alpine/edge/testing geos geos-dev
RUN go get github.com/paulsmith/gogeos/geos \
    && go get github.com/davecgh/go-spew/spew \
    && go get github.com/go-redis/redis \
    && go get github.com/jinzhu/gorm \
    && go get github.com/lib/pq

RUN go build -o /app/poller

CMD [ "/app/poller" ]
