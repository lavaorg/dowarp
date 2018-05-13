// Copyright 2018 Larry Rau. All rights reserved
// See Apache2 LICENSE

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/lavaorg/lrt/mlog"
	"github.com/lavaorg/warp/warp9"
)

var verbose = flag.Bool("v", false, "verbose mode")
var dbglev = flag.Int("d", 0, "debuglevel")
var addr = flag.String("a", "127.0.0.1:9901", "network address")
var aname = flag.String("aname", ".", "path on server to use as root")
var alt = flag.String("alt", "192.168.1.112", "alternate reporting address")

func main() {

	flag.Parse()

	// accept a network connection
	listenForServers("tcp", *addr)

	return
}

func usage() {
	fmt.Println("usage: np [-v][-d dbglev] [-a addr] cmd arg")
	fmt.Println("\tcmd = {ls,stat,cat,echo}")
}

// list on the indicated network and address and then mount the calling server.
// This is a reverse mount; the server iniitates a connection and then this
// client performs the mount on that conneciton.
func listenForServers(ntype, addr string) {

	l, err := net.Listen(ntype, addr)
	if err != nil {
		mlog.Error("listen failed:%v", err)
	}

	for {
		c, err := l.Accept()
		if err != nil {
			mlog.Error("accept fail:v", err)
			return
		}
		mlog.Debug("accepted connection: %v", c)
		go handleConnection(c)
	}
}

// mount the server that just called on the Conn
func handleConnection(c net.Conn) {

	warp9.DefaultDebuglevel = *dbglev
	uid := uint32(0xFFFFFFFF & uint32(os.Getuid()))
	user := warp9.Identity.User(uid)

	c9, err := warp9.MountConn(c, *aname, 500, user)
	if err != warp9.Egood {
		mlog.Error("Error:%v\n", err)
		c.Close()
		return // end thread
	}

	// read target sensor
	readSensor(c9)

	// reconfigure the target if necessar
	reconfigSensor(c9)

	// close connection
	c9.Clunk(c9.Root)
	c9.Unmount()
}

func readSensor0(c9 *warp9.Clnt) {
	f, err := c9.Open("sensors", warp9.OREAD)
	if err != warp9.Egood {
		log.Fatalf("Error:%v\n", err)
	}
	defer f.Close()

	buf := make([]byte, 8192)
	for {
		n, err := f.Read(buf)
		if n == 0 {
			break
		}
		if err != warp9.Egood {
			log.Fatalf("Error reading:%v\n", err)
		}
		mlog.Info("%v", string(buf))
		if err == warp9.Eeof {
			break
		}
	}

	if err != warp9.Egood && err != warp9.Eeof {
		mlog.Error("error:%v", err)
		return
	}

}

// this shortens the number of requests due to avoiding a
// last read that just looks for EOF. We have knowledge that
// the sensors being read is a small number of bytes.
func readSensor(c9 *warp9.Clnt) {

	fid, err := c9.Walk("sensors")
	if err != warp9.Egood {
		mlog.Error("could not Walk:%v", err)
		return
	}
	defer c9.Clunk(fid)
	err = c9.FOpen(fid, warp9.OREAD)
	if err != warp9.Egood {
		mlog.Error("open failed:%v", err)
		return
	}

	buf, err := c9.Read(fid, uint64(0), uint32(100))
	if err != warp9.Egood {
		mlog.Error("Error:%v\n", err)
	} else {
		mlog.Info("%v", string(buf))
	}
}

var reportCount = 0

// every 10 readings we will reconfigure the sensor to report
// somewere else.
func reconfigSensor(c9 *warp9.Clnt) {
	reportCount++

	if reportCount > 2 {
		reportCount = 0
		fid, err := c9.Walk("ctl")
		if err != warp9.Egood {
			mlog.Error("could not Walk:%v", err)
			return
		}
		defer c9.Clunk(fid)
		err = c9.FOpen(fid, warp9.OWRITE)
		if err != warp9.Egood {
			mlog.Error("open failed:%v", err)
			return
		}
		data := []byte("ip:" + *alt)
		cnt, e := c9.Write(fid, data, 0)
		if e != nil {
			mlog.Error("Error:%v\n", err)
		} else {
			mlog.Info("ctl: ip:%v [%d]", alt, cnt)
		}
	}
}
