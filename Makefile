VERSION=${WAF_VERSION}

.PHONY: build
build:
	go build -o bin/api cmd/main.go

.PHONY: run
run: build
	./bin/api

.PHONY: image
image:
	docker build . -t arthurcgc/waf:$(VERSION)
	docker push arthurcgc/waf:$(VERSION)

.PHONY: rbac
rbac:
	kubectl apply -f k8s/rbac/

.PHONY: deploy
deploy:
	kubectl apply -f k8s/deploy.yaml

.PHONY: deploy-vulnerable
deploy-vulnerable:
	kubectl apply -f k8s/vulnerable-web-app/dvwa.yaml

.PHONY: all
all: build image rbac
	kubectl apply -f k8s/deploy.yaml