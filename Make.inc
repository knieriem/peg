ifeq ($(PEGDIR),)
PEGDIR := $(subst /Make.inc,,$(lastword $(MAKEFILE_LIST)))
endif
PEG=$(PEGDIR)/cmd/peg/peg
LEG=$(PEGDIR)/cmd/leg/leg
BOOTSTRAP=$(PEGDIR)/cmd/peg/bootstrap.go

$(PEG):	$(BOOTSTRAP)
	cd $(@D) && GOARCH= GOOS= go build

$(LEG):	$(PEGDIR)/cmd/leg/leg.go
	cd $(@D) && GOARCH= GOOS= go build

$(BOOTSTRAP):
	cd $(PEGDIR) && GOARCH= GOOS= go install
	cd $(@D) && go run ../../bootstrap/main.go > $(@F)

%.go: %.peg $(PEG)
	$(PEG) -switch -inline -O all $< > $@
