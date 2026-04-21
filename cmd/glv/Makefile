SHELL := /bin/bash

%: # Silence errors about non existing targets
	@true

docs:
	@notatio coi --command="${BASE_NAME} glv parse --help" --document-path=README.md --header='`parse`' --limiter-left=### --limiter-right="### " --index=1
	@notatio coi --command="${BASE_NAME} glv show --help" --document-path=README.md --header='`show`' --limiter-left=### --limiter-right="## " --index=1
