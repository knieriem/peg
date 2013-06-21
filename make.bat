@echo off
go install
go run bootstrap/main.go > cmd/peg/bootstrap.go
go build ./cmd/peg
for %%f in (cmd/leg/leg calculator/calculator) do peg -switch -inline -O all %%f.peg > %%f.go
go build ./cmd/leg
for %%f in (cmd/legleg/leg cmd/legcalc/calc) do leg -switch -O all %%f.leg > %%f.go
