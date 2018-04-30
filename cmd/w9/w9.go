// Copyright 2018 Larry Rau. All rights reserved
// See Apache2 LICENSE

package main

// simple wrapper for running object servers listening on local network port

import (
	"flag"
	"log"

	_ "github.com/lavaorg/lrt/mlog"

	"github.com/lavaorg/dowarp/w9"
	"github.com/lavaorg/warp/warp9"
)

var addr = flag.String("a", ":5640", "network address")
var debug = flag.Int("debug", 0, "print debug messages")
var oserver = flag.String("s", "echoos", "object server")

func main() {
	flag.Parse()
	oserv := w9.StartServer("w9", 1)
	root := oserv.GetRoot()
	i, _ := root.Create("wow", warp9.DMAPPEND, 0)
	o := i.(*w9.OneItem)
	o.Buffer = []byte("Wow!")
	root.Create("big", warp9.DMTMP, 0)
	err := oserv.StartNetListener("tcp", *addr)
	if err != nil {
		log.Println(err)
	}
}
