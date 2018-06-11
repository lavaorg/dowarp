// Copyright 2018 Larry Rau. All rights reserved
// See Apache2 LICENSE

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/lavaorg/lrt/mlog"
	"github.com/lavaorg/warp/warp9"
)

var verbose = flag.Bool("v", false, "verbose mode")
var dbglev = flag.Int("d", 0, "debuglevel")
var addr = flag.String("a", "127.0.0.1:9901", "network address")
var aname = flag.String("aname", ".", "path on server to use as root")
var alt = flag.String("alt", "", "alternate reporting address")

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

// listen on the indicated network and address and then mount the calling server.
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
	if err != nil {
		mlog.Error("Mount failure:%v", err)
		c.Close()
		return // end thread
	}

	// read target sensor
	readSensor(c9)

	// reconfigure the target if necessar
	if *alt != "" {
		reconfigSensor(c9)
	}

	// close connection
	c9.Clunk(c9.Root)
	c9.Unmount()
	if prf_reads > 9980 {
		fmt.Printf("prf: %v reads %v time. %v read/sec. %v msg/sec\n",
			prf_reads, prf_time, (float64(prf_reads) / prf_time.Seconds()), (float64(prf_reads*15) / prf_time.Seconds()))
		//memstats()
	}
}

func readSensor0(c9 *warp9.Clnt) {
	f, err := c9.Open("sensors", warp9.OREAD)
	if err != nil {
		log.Fatalf("Error:%v\n", err)
	}
	defer f.Close()

	buf := make([]byte, 8192)
	for {
		n, err := f.Read(buf)
		if n == 0 {
			break
		}
		if err != nil {
			log.Fatalf("Error reading:%v\n", err)
		}
		mlog.Info("%v", string(buf))
		if err == warp9.WarpErrorEOF {
			break
		}
	}

	if err != nil && err != warp9.WarpErrorEOF {
		mlog.Error("error:%v", err)
		return
	}

}

var prf_reads int64
var prf_time time.Duration

// this shortens the number of requests due to avoiding a
// last read that just looks for EOF. We have knowledge that
// the sensors being read is a small number of bytes.
func readSensor(c9 *warp9.Clnt) {

	fid, err := c9.Walk("sensors")
	if err != nil {
		mlog.Error("could not Walk:%v", err)
		return
	}
	defer c9.Clunk(fid)
	err = c9.FOpen(fid, warp9.OREAD)
	if err != nil {
		mlog.Error("open failed:%v", err)
		return
	}
	t0 := time.Now()
	buf, err := c9.Read(fid, uint64(0), uint32(100))
	prf_reads++
	t1 := time.Now()
	prf_time = prf_time + t1.Sub(t0)

	if err != nil {
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
		if err != nil {
			mlog.Error("could not Walk:%v", err)
			return
		}
		defer c9.Clunk(fid)
		err = c9.FOpen(fid, warp9.OWRITE)
		if err != nil {
			mlog.Error("open failed:%v %v", fid, err)
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

// obtain the object servers memory stats
func memstats() {
	var ms runtime.MemStats

	runtime.ReadMemStats(&ms)
	//fmt.Printf("ms:%v\n", ms)
	fmt.Printf("Sys:\t\t%v\n", ms.Sys)
	fmt.Printf("HeapAlloc:\t%v\n", ms.HeapAlloc)
	fmt.Printf("Mallocs:\t%v\n", ms.Mallocs)
	fmt.Printf("Frees:\t\t%v\n", ms.Frees)
	fmt.Printf("HeapSys:\t%v\n", ms.HeapSys)
	fmt.Printf("HeapIdle:\t%v\n", ms.HeapIdle)
	fmt.Printf("HeapInuse:\t%v\n", ms.HeapInuse)
	fmt.Printf("HeapReleased:\t%v\n", ms.HeapReleased)
	fmt.Printf("HeapObjects:\t%v\n", ms.HeapObjects)
	fmt.Printf("StackInuse:\t%v\n", ms.StackInuse)
	fmt.Printf("StackSys:\t%v\n", ms.StackSys)
	fmt.Printf("MSpanInuse:\t%v\n", ms.MSpanInuse)
	fmt.Printf("MSpanSys:\t%v\n", ms.MSpanSys)
	fmt.Printf("MCacheInuse:\t%v\n", ms.MCacheInuse)
	fmt.Printf("MCacheSys:\t%v\n", ms.MCacheSys)
	fmt.Printf("BuckHashSys:\t%v\n", ms.BuckHashSys)
	fmt.Printf("GCSys:\t\t%v\n", ms.GCSys)
	fmt.Printf("OtherSys:\t%v\n", ms.OtherSys)
	fmt.Printf("GCCPUFraction:\t%v\n", ms.GCCPUFraction)

	/*
		pat := "********************"
		fmt.Print("Mallocs by object size:\n")
		for _, s := range ms.BySize {
			p := int((float32(s.Mallocs) / float32(ms.Mallocs)) * 20.0)
			fmt.Printf("%d:%d\t:%v\n", s.Size, s.Mallocs, pat[:p])
		}
	*/
	return
}
