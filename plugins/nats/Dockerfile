FROM golang:1.12.5-alpine3.9 AS FirstStage
MAINTAINER Anand Sundararaj <Anand.S.Main@gmail.com>

WORKDIR $GOPATH/src/github.com/nats-io
RUN apk add --update git
COPY . .
RUN git clone http://github.com/nats-io/nats-server
WORKDIR $GOPATH/src/github.com/nats-io/nats-server
RUN CGO_ENABLED=0 GO111MODULE=off GOOS=linux go build -o /nats-server

FROM scratch
COPY ./hybridpipe_nats.conf /hybridpipe_nats.conf
COPY --from=FirstStage /nats-server /nats-server
EXPOSE 4222 4442 5222 6332 6222 8222 8884

ENTRYPOINT ["/nats-server"]
CMD ["-c", "hybridpipe_nats.conf"]
