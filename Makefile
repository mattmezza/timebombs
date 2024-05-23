.PHONY: help
help: ## show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[$$()% a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


.PHONY: format lint
format:  ## format the code with black and isort
	autoflake --remove-all-unused-imports --recursive --remove-unused-variables --in-place src tests --exclude=__init__.py
	black src tests
	isort src tests
lint:  ## lints src code with mypy, black and isort (--check-only)
	mypy src
	black src tests --check
	isort src tests --check-only

.PHONY: docs docs-live
docs:  ## builds the docs website
	python -m mkdocs build
docs-live: docs  ## builds and serves the docs website
	python -m mkdocs serve --dev-addr 127.0.0.1:8008

.PHONY: clean
clean:  ## cleans dist and site folders
	@rm -rf dist
	@rm -rf site

.PHONY: test
test:  ## runs the tests with coverage report
	pytest --cov=timebombs --cov-report=term-missing --cov-report=xml -o console_output_style=progress tests
