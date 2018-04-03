.PHONY: all

all:
	./check.sh
	pip install pycodestyle pep257
	cd ../utils/ && pycodestyle *.py
	cd ../utils/ && pep257 *.py
