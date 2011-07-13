# Copyright 2010 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

PKGROOT = .
include Make.common

.PHONY: all
all: peg

peg: main.$(O)
	$(LD) -o peg main.$(O)

PKGFILES=\
	peg.go\

CMDFILES=\
	main.go\
	bootstrap.go\

peg.$(O):\
	$(PKGFILES)\

main.$(O):\
	peg.$(O)\
	$(CMDFILES)\

bootstrap.go: bootstrap/main.go
	$(MAKE) -C bootstrap/ bootstrap
	./bootstrap/bootstrap

.PHONY: clean
clean:
	$(MAKE) -C bootstrap/ clean
	$(MAKE) -C leg/ clean
	rm -f *.6 *.8 bootstrap.go peg

.PHONY: test
test:	peg
	./peg -inline -switch peg.peg
	cmp peg.peg.go bootstrap.go
