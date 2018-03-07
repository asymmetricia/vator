FROM debian:stable

ADD vator /vator
ADD run.sh /run.sh

ENTRYPOINT /run.sh
