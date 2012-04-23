PEGDIR=.

PARSERGOFILES=\
	./calculator/calculator.go\
	./cmd/leg/leg.go\
	\
	./cmd/legleg/leg.go\
	./cmd/legcalc/calc.go\

all:	prepare

include Make.inc
include cmd/leg/Make.inc

prepare:	$(PEG) $(PARSERGOFILES)

clean:
	go clean ./...
	rm -f $(BOOTSTRAP)
	rm -f $(PARSERGOFILES)

# compared files must be equal
test:	./cmd/peg/peg.peg.go
	diff $(<D)/bootstrap.go $<
	rm -f $<

.PHONY:\
	all\
	prepare\
	clean\
