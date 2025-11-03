.PHONY: wire
# generate wire
wire:
	find apps -type d -depth 1 -print | xargs -L 1 bash -c 'cd "$$0" && pwd && $(MAKE) wire'

.PHONY: api
# generate api
api:
	find apps -type d -depth 1 -print | xargs -L 1 bash -c 'cd "$$0" && pwd && $(MAKE) api'

# generate proto
proto:
	find apps -type d -depth 1 -print | xargs -L 1 bash -c 'cd "$$0" && pwd && $(MAKE) proto'

genmocks:
	mockgen -source=./apps/teller/internal/biz/notification.go -destination=./apps/teller/internal/biz/notification_mock_test.go -package=biz
	mockgen -source=./apps/teller/internal/biz/provider.go -destination=./apps/teller/internal/biz/provider_mock_test.go -package=biz
.PHONY: tidy-modules
tidy-modules:
	@find . -type d \( -name build -prune \) -o -name go.mod -print | while read -r gomod_path; do \
		dir_path=$$(dirname "$$gomod_path"); \
		echo "Executing 'go mod tidy' in directory: $$dir_path"; \
		(cd "$$dir_path"  && GOPROXY=$(GOPROXY) go get -u ./... && GOPROXY=$(GOPROXY) go mod tidy) || exit 1; \
	done
