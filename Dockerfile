# syntax=docker/dockerfile:1-labs
FROM golang:1

WORKDIR /work

COPY --parents cmds *.go go.mod go.sum log models static templates /work/

RUN go get .
RUN go build -o /vator .

FROM debian:stable

RUN export DEBIAN_FRONTEND=noninteractive; \
    apt-get update; \
    apt-get -y install ca-certificates tini

COPY --from=0 /vator /vator
COPY run.sh /

ENTRYPOINT [ "tini", "--", "/run.sh" ]
