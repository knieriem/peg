# Copyright 2010 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

include ../Make.common

.PHONY: all
all: leg

leg: main.$(O)
	$(LD) -o leg main.$(O)

main.$(O): leg.$(O)

leg.$(O): leg.go leg.peg.go
	$(GC) $^

leg.go: ../peg.go
	sed 's,^package peg,package leg,g' < $< > $@

%.peg.go: %.peg ../peg
	../peg -inline -switch $<

.PHONY: clean
clean:
	rm -f *.6 *.8 leg *.peg.go leg *leg.go calc

%: %.leg leg
	./leg $<
	$(GC) $<.go
	$(LD) -o $@ $<.$(O)
