SRC=$(shell find . -type f | grep "\.py$$" | grep -v "\_\_init\_\_.py")
.PHONY: tests
all: tests analyze

tests:
	cd tests && make -C .

analyze:
	pip install pycodestyle pep257
	pycodestyle $(SRC)
	pep257 $(SRC)
