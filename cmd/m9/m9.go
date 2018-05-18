// Copyright 2018 Larry Rau. All rights reserved
// See Apache2 LICENSE

package main

// simple wrapper for running object servers listening on local network port

import (
	"flag"

	"github.com/lavaorg/lrt/mlog"

	"github.com/lavaorg/warp/warp9"
	"github.com/lavaorg/warp/wkit"
)

var addr = flag.String("a", ":9091", "network address")
var debug = flag.Int("d", 0, "print debug messages")
var oserver = flag.String("s", "echoos", "object server")

const PermUGO = 0x1A0 //rw- r-- ---
const PermRO = 0x120  //r-- r-- ---

func main() {
	flag.Parse()

	// create our object server
	oserv := wkit.StartServer("w9", *debug)
	root := oserv.GetRoot()

	// create two objects to be served that just can store bytes
	// in ram; bytes can be read or written
	i, _ := root.CreateItem(nil, "data", warp9.DMAPPEND|PermUGO)
	o := i.(*wkit.OneItem)
	o.Buffer = []byte("Wow!")

	// create a mount point and add it
	usr := warp9.Identity.User(1)
	mt, err := wkit.MountPointDial("tcp", "localhost:5640", "/", 0, usr)
	if err != nil {
		mlog.Error("could not mount:%v", err)
		return
	}
	mt.Debug(*debug)
	err = root.AddItem(mt, "mnt")
	if err != nil {
		mlog.Error("could not add mount", err)
		return
	}
	// start serving
	err = oserv.StartNetListener("tcp", *addr)
	if err != nil {
		mlog.Error("could not start listening:%v", err)
	}
}
