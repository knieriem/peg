# to be included after ../../Make.inc

%.go: %.leg $(LEG)
	$(LEG) -switch -O all $< > $@
