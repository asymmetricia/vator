FROM debian:stable

RUN export DEBIAN_FRONTEND=noninteractive; \
    apt-get update; \
    apt-get -y install ca-certificates

ADD vator /vator
ADD run.sh /run.sh

ENTRYPOINT /run.sh
