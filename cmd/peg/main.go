// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/knieriem/peg"
	"io/ioutil"
	"log"
	"os"
	"runtime"
)

var (
	inline    = flag.Bool("inline", false, "parse rule inlining")
	_switch   = flag.Bool("switch", false, "replace if-else if-else like blocks with switch blocks")
	optiFlags = flag.String("O", "", "turn on various optimizations")
)

func main() {
	runtime.GOMAXPROCS(2)
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		fmt.Fprintf(os.Stderr, "  FILE: the peg file to compile\n")
		os.Exit(1)
	}
	file := flag.Arg(0)

	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	p := &Peg{Tree: peg.New(*inline, *_switch), Buffer: string(buffer)}
	p.Init()
	if err = p.Parse(0); err == nil {
		w := bufio.NewWriter(os.Stdout)
		p.Compile(w, *optiFlags)
		w.Flush()
	} else {
		log.Print(file, ":", err)
	}
}
