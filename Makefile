.PHONY: format docu docu-live clean lint test

format:
	autoflake --remove-all-unused-imports --recursive --remove-unused-variables --in-place src tests --exclude=__init__.py
	black src tests
	isort src tests

docu:
	cp README.md docs/index.md
	python -m mkdocs build

docu-live: docu
	python -m mkdocs serve --dev-addr 127.0.0.1:8008

clean:
	@rm -rf dist
	@rm -rf site

lint:
	mypy src
	black src tests --check
	isort src tests --check-only

test:
	pytest --cov=timebombs --cov-report=term-missing --cov-report=xml -o console_output_style=progress tests
