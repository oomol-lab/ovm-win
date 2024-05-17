.PHONY: all build build-amd64 force-build clean help

all: help

##@
##@ Build commands
##@

build: ##@ Build binaries for all architectures
	@$(MAKE) out/ovm-amd64

build-amd64: ##@ Build amd64 binary
	@$(MAKE) out/ovm-amd64

out/ovm-amd64: out/ovm-%: force-build
	@mkdir -p $(@D)
	GOOS=windows GOARCH=$* go build -o $@.exe ./cmd/ovm

force-build:


##@
##@ Clean commands
##@

clean: ##@ Clean up build artifacts
	$(RM) -rf out


##@
##@ Misc commands
##@

help: ##@ (Default) Print listing of key targets with their descriptions
	@printf "\nUsage: make <command>\n"
	@grep -F -h "##@" $(MAKEFILE_LIST) | grep -F -v grep -F | sed -e 's/\\$$//' | awk 'BEGIN {FS = ":*[[:space:]]*##@[[:space:]]*"}; \
	{ \
		if($$2 == "") \
			printf ""; \
		else if($$0 ~ /^#/) \
			printf "\n%s\n", $$2; \
		else if($$1 == "") \
			printf "     %-20s%s\n", "", $$2; \
		else \
			printf "\n    \033[34m%-20s\033[0m %s\n", $$1, $$2; \
	}'
