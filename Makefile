.PHONY: clean tail

IMAGE_URL ?= 248174752766.dkr.ecr.us-west-1.amazonaws.com/vator

restart: .push
	ssh admin@mapbot.cernu.us sudo systemctl restart vator-dev

push: .push
.push: .docker
	@ set -e; \
	eval "$$(aws --region=us-west-1 ecr get-login --no-include-email)" && \
	docker push ${IMAGE_URL}-dev && \
	touch .push

.docker: vator Dockerfile run.sh
	docker build --pull -t vator .
	docker tag vator ${IMAGE_URL}-dev
	touch .docker

restart-prod: .push-prod
	ssh admin@mapbot.cernu.us sudo systemctl restart vator

push-prod: .push-prod
.push-prod: .docker-prod
	@ set -e; \
	eval "$$(aws --region=us-west-1 ecr get-login --no-include-email)" && \
	docker push ${IMAGE_URL} && \
	touch .push-prod

.docker-prod: vator Dockerfile run.sh
	docker build --pull -t vator .
	docker tag vator ${IMAGE_URL}
	touch .docker-prod

vator: ${shell find -name \*.go}
	go fmt github.com/pdbogen/vator/...
	go build -o vator

tail:
	ssh admin@mapbot.cernu.us journalctl -u vator-dev -f
tail-prod:
	ssh admin@mapbot.cernu.us journalctl -u vator -f
