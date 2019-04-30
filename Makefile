.PHONY: clean tail

IMAGE_URL ?= 248174752766.dkr.ecr.us-west-1.amazonaws.com/vator

restart: .push
	ssh core@mapbot.cernu.us sudo systemctl restart vator

push: .push
.push: .docker
	@ set -e; \
	eval "$$(aws --region=us-west-1 ecr get-login --no-include-email)" && \
	docker push ${IMAGE_URL} && \
	touch .push

.docker: vator Dockerfile run.sh
	docker build -t vator .
	docker tag vator ${IMAGE_URL}
	touch .docker

vator: ${shell find -name \*.go}
	go fmt github.com/pdbogen/vator/...
	go build -o vator

tail:
	ssh core@mapbot.cernu.us journalctl -u vator -f
