.PHONY: clean tail

IMAGE_URL ?= 248174752766.dkr.ecr.us-west-1.amazonaws.com/vator

restart: .docker
	ssh admin@mapbot.cernu.us sudo systemctl restart vator-dev

.docker: Dockerfile run.sh ${shell find -name \*.go} static/js/graph.js
	@ set -e; \
	eval "$$(aws --region=us-west-1 ecr get-login --no-include-email)" && \
	docker buildx build --push --platform linux/amd64,linux/arm64 --pull -t ${IMAGE_URL}-dev .
	touch .docker

.docker_prod:
	@ set -e; \
	eval "$$(aws --region=us-west-1 ecr get-login --no-include-email)" && \
	docker buildx build --push --platform linux/amd64,linux/arm64 --pull -t ${IMAGE_URL} .
	touch .docker_prod
	
restart-prod: .docker_prod
	ssh admin@mapbot.cernu.us sudo systemctl restart vator

tail:
	ssh admin@mapbot.cernu.us journalctl -u vator-dev -f
tail-prod:
	ssh admin@mapbot.cernu.us journalctl -u vator -f

reset-dev:
	ssh admin@mapbot.cernu.us sudo rm -f /opt/vator-dev/vator.db
	ssh admin@mapbot.cernu.us sudo systemctl restart vator-dev

static/js/graph.js: ${shell find static/js -name \*.ts}
	./build-js.sh

clean:
	$(RM) .docker .docker_prod static/js/graph.js
