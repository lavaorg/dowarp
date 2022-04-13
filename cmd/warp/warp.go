package main

import (
	"flag"
	"fmt"

	"log"
	"os"
	"time"

	"github.com/lavaorg/warp/tools"
	"github.com/lavaorg/warp/warp9"
)

var Dbglev *int
var Addr *string
var Aname *string

func init() {
	Dbglev = flag.Int("d", 0, "debuglevel")
	Addr = flag.String("a", ":9090", "network address")
	Aname = flag.String("aname", "/", "path on server to use as root")
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
		log.Fatal("expected an argument")
	}

	//uid := uint32(0xFFFFFFFF & uint32(os.Getuid()))
	user := warp9.Identity.User(1)
	warp9.DefaultDebuglevel = *Dbglev
	//warp9.LogDebug(true)

	c9, err := warp9.Mount("tcp", *Addr, *Aname, 8192, user)
	if err != nil {
		werr := err.(*warp9.WarpError)
		if werr != nil {
			log.Fatalf("Error:%v: (addr=%v aname=%v)\n", err, *Addr, *Aname)
		}
	}
	defer c9.Clunk(c9.Root)

	args := flag.Args()
	cmd := args[0]
	args = args[1:]

	switch cmd {

	default:
		{
			usage()
			log.Fatal("unknown cmd")
		}

	case "cat":
		tools.Cat(c9, args[0])
	case "ctl":
		tools.Ctl(c9, args)
	case "ls":
		tools.Ls(c9, args[0])
	case "stat":
		tools.Stat(c9, args[0])
	case "get":
		tools.Get(c9, args[0])
	case "write":
		tools.Write(c9, args[0])
	case "dcat":
		dcat(c9, args[0])
	}
	return
}

// dump usage at commandline.
func usage() {
	var err error
	o := flag.CommandLine.Output()
	msg := func(m string) {
		if err != nil {
			return
		}
		_, err = fmt.Fprintf(o, m)
	}

	msg("warp:\tsimple command line warp9 client to manipulate objects.\n")
	msg("Usage of warp: [-v][-d dbglev] [-a addr] cmd arg\n")
	flag.PrintDefaults()
	msg("\n  cmd = {ls,stat,cat,get,write,ctl}\n")
	msg("\tls - list contents of a directory object\n")
	msg("\tstat - show metadata of an object\n")
	msg("\tcat - copy contents of an object to stdout\n")
	msg("\tget - perform a get (open/read/close) operation (like cat)\n")
	msg("\twrite - copy contents of stdin to object (use: echo hello| np write)\n")
	msg("\tctl - perform a write of rest of args to object; then read contents and write to stdout\n")
	msg("\tdcat - copy contents of an object to stdout, sleep 10 before closing\n")
	if err != nil {
		log.Fatalf("usage message failed: %v", err)
	}
}

func dcat(c9 *warp9.Clnt, obj string) {
	o, err := c9.Open(obj, warp9.OREAD)
	if err != nil {
		log.Fatalf("Error:%v\n", err)
	}
	defer o.Close()

	buf := make([]byte, 8192)

	for {
		n, err := o.Read(buf)
		if n == 0 {
			break
		}
		if err != nil && err != warp9.WarpErrorEOF {
			warp9.Error("Error reading:%v\n", err)
		}
		_, _ = os.Stdout.Write(buf[0:n])
		if err == warp9.WarpErrorEOF {
			break
		}
	}

	if err != nil && err != warp9.WarpErrorEOF {
		log.Fatalf("Error:%v\n", err)
	}
	time.Sleep(10 * time.Second)
}
