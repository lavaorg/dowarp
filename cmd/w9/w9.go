// Copyright 2018 Larry Rau. All rights reserved
// See Apache2 LICENSE

package main

import "flag"

/*
create a very basic warp9 server that listens on local network

Server will export a directory structure:
  /ctl      -- command object: write command; read results
  /wow      -- read/write string
  /big      -- read/write string
  /sensors
	/temp1  -- dynamic value
  	/temp2
 	/tmp3
*/

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"

	"github.com/lavaorg/warp/wkit"

	"github.com/lavaorg/dowarp/fakesen"
)

var addr = flag.String("a", ":9090", "network address")
var debug = flag.Int("d", 0, "print debug messages")
var oserver = flag.String("s", "echoos", "object server")

const PermUGO = 0x1A0 //rw- r-- ---
const PermRO = 0x120  //r-- r-- ---

func main() {
	flag.Parse()

	// create objects
	ctl := makeCtl()

	// create a read-only string item
	wow := wkit.NewItem("wow")
	wow.SetBuffer([]byte("Wow!"))
	wow.SetMode(PermRO)

	// create another read-only string object
	hello := wkit.NewItem("hello")
	hello.SetBuffer([]byte("Hello World!"))
	hello.SetMode(PermRO)

	// create a sub-directory with fake temp sensors
	sensors := makeSensors()

	// arrange objects
	root := wkit.NewDirItem("/")
	root.AddItem(ctl)
	root.AddItem(wow)
	root.AddItem(hello)
	root.AddDirectory(sensors)

	// create server
	srv := wkit.NewServer("w9", *debug, root)
	srv.Start(srv)

	// start serving
	err := srv.StartNetListener("tcp", *addr)
	if err != nil {
		log.Println(err)
	}
}

func makeSensors() wkit.Directory {
	sdir := wkit.NewDirItem("sensors")
	sdir.AddItem(fakesen.NewFakeSensor("temp1"))
	sdir.AddItem(fakesen.NewFakeSensor("temp2"))
	sdir.AddItem(fakesen.NewFakeSensor("temp3"))
	return sdir
}

// create the single "Command" object called "ctl" and
// create a number of methods that can be invoked by writing
// the method and its parameters and then reading the results.
// Command objects are inherently "Append" only
// Normal procedure is to perform a Write/Read pari and thus open
// the object in O_RDWR mode.
//
func makeCtl() wkit.Item {
	cmd := wkit.NewCommand("ctl", nil, &MyCtl{"larry\n"})
	cmd.AddMethod("hello", hello)
	cmd.AddMethod("add", add)
	cmd.AddMethod("cpus", cpus)
	cmd.AddMethod("memstats", memstats)
	return cmd
}

type MyCtl struct {
	msg string
}

//
// example methods of the ctl oject
//

// say hello; get a response
func hello(ctx wkit.CmdCtx, cmd *wkit.Command, cmdname string, args []byte) error {
	myctl, ok := ctx.(*MyCtl)
	if !ok {
		return errors.New("cmdobj:bad ctx")
	}

	if cmdname != "hello" {
		return errors.New("bad command item")
	}

	cmd.SetBuffer([]byte(myctl.msg))
	return nil
}

// add a seqence of numbers represented in text form space separated
func add(ctx wkit.CmdCtx, cmd *wkit.Command, cmdname string, args []byte) error {
	s := string(args)
	aa := strings.Split(s, " ")

	r := 0
	for _, a := range aa {
		n, e := strconv.Atoi(a)
		if e != nil {
			return errors.New("bad math")
		}
		r = r + n
	}
	cmd.SetBuffer([]byte(strconv.Itoa(r) + "\n"))
	return nil
}

// discover the object server's number of virtual CPUs
func cpus(ctx wkit.CmdCtx, cmd *wkit.Command, cmdname string, args []byte) error {
	cmd.SetBuffer([]byte(strconv.Itoa(runtime.NumCPU()) + "\n"))
	return nil
}

// obtain the object servers memory stats
func memstats(ctx wkit.CmdCtx, cmd *wkit.Command, cmdname string, args []byte) error {
	var ms runtime.MemStats
	var b bytes.Buffer

	runtime.ReadMemStats(&ms)
	//fmt.Fprintf(&b, "ms:%v\n", ms)
	fmt.Fprintf(&b, "Sys:\t\t%v\n", ms.Sys)
	fmt.Fprintf(&b, "HeapAlloc:\t%v\n", ms.HeapAlloc)
	fmt.Fprintf(&b, "Mallocs:\t%v\n", ms.Mallocs)
	fmt.Fprintf(&b, "Frees:\t\t%v\n", ms.Frees)
	fmt.Fprintf(&b, "HeapSys:\t%v\n", ms.HeapSys)
	fmt.Fprintf(&b, "HeapIdle:\t%v\n", ms.HeapIdle)
	fmt.Fprintf(&b, "HeapInuse:\t%v\n", ms.HeapInuse)
	fmt.Fprintf(&b, "HeapReleased:\t%v\n", ms.HeapReleased)
	fmt.Fprintf(&b, "HeapObjects:\t%v\n", ms.HeapObjects)
	fmt.Fprintf(&b, "StackInuse:\t%v\n", ms.StackInuse)
	fmt.Fprintf(&b, "StackSys:\t%v\n", ms.StackSys)
	fmt.Fprintf(&b, "MSpanInuse:\t%v\n", ms.MSpanInuse)
	fmt.Fprintf(&b, "MSpanSys:\t%v\n", ms.MSpanSys)
	fmt.Fprintf(&b, "MCacheInuse:\t%v\n", ms.MCacheInuse)
	fmt.Fprintf(&b, "MCacheSys:\t%v\n", ms.MCacheSys)
	fmt.Fprintf(&b, "BuckHashSys:\t%v\n", ms.BuckHashSys)
	fmt.Fprintf(&b, "GCSys:\t\t%v\n", ms.GCSys)
	fmt.Fprintf(&b, "OtherSys:\t%v\n", ms.OtherSys)
	fmt.Fprintf(&b, "GCCPUFraction:\t%v\n", ms.GCCPUFraction)

	pat := "********************"
	fmt.Fprint(&b, "Mallocs by object size:\n")
	for _, s := range ms.BySize {
		p := int((float32(s.Mallocs) / float32(ms.Mallocs)) * 20.0)
		fmt.Fprintf(&b, "%d:%d\t:%v\n", s.Size, s.Mallocs, pat[:p])
	}

	cmd.SetBuffer(b.Bytes())
	return nil
}
