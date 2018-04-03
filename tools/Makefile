SRC=$(shell find . -type f | grep "\.py$$")
all: tests analyze

tests:
	cd tests && make -C .

analyze:
	pip install pycodestyle pep257
	pycodestyle $(SRC)
	pep257 $(SRC)
