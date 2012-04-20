# to be included after ../../Make.inc

%.leg.go: %.leg $(LEG)
	$(LEG) -switch $<
