vet:
	go vet ./...

staticcheck:
	staticcheck ./...

install-staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck@latest

check-build:
	go build -o concat-video .

check-formatting:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "The following files need formatting:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
