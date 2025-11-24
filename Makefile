.PHONY:	benc

.PHONY:	fmt
fmt:
	@sh ./.script/goimports.sh

.PHONY: tidy
tidy:
	@go mod tidy -v

.PHONY: check
check:
	@$(MAKE) fmt
	@$(MAKE) tidy
