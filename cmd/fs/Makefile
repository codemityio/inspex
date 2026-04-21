SHELL := /bin/bash

%: # Silence errors about non existing targets
	@true

docs:
	@notatio coi --command="${BASE_NAME} fs scan --help" --document-path=README.md --header='`scan`' --limiter-left=### --limiter-right="### " --index=1
	@notatio coi --command="${BASE_NAME} fs table --help" --document-path=README.md --header='`table`' --limiter-left=### --limiter-right="## " --index=1
