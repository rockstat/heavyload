NAME := heavyload
REPO := rockstat/$(NAME)
LOCATION := /go/src/heavyload

# run-dev:
# 	docker build -f dev/Dockerfile -t heavyload-dev .
# 	docker run --rm -it \
# 		-v $$PWD:$(LOCATION) \
# 		-p 10010:8080 \
# 		-e WEBHOOK=http://docker.for.mac.localhost:10001/{{.service}}/{{.name}} \
# 		--network custom \
# 		--name heavyload-dev \
# 		--hostname heavyload-dev \
# 		heavyload-dev

run:
	docker run --rm -it \
		-v $$PWD:$(LOCATION) \
		-p 10010:8080 \
		-e WEBHOOK=http://docker.for.mac.localhost:10001/{{.service}}/{{.name}} \
		--network custom \
		--name $(NAME) \
		--hostname $(NAME) \
		$(NAME)

# bump-patch:
# 	bumpversion patch

# bump-minor:
# 	bumpversion minor

# build-dev:
# 	docker build -t $(NAME):dev .

# push-dev:
# 	docker tag $(NAME) $(REPO):dev
# 	docker push $(REPO):dev


push-latest:
	docker tag $(NAME) $(REPO):latest
	docker push $(REPO):latest


build_image:
	docker build -t $(NAME) .


build_amd64:
	docker buildx build --platform linux/amd64 -t $(NAME) .

