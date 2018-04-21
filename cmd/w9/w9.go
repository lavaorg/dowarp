// Copyright 2018 Larry Rau. All rights reserved
// See Apache2 LICENSE

package main

// simple wrapper for running object servers listening on local network port

import (
	"flag"
	"fmt"
	"log"

	"github.com/lavaorg/dowarp/nullfs"
	//"github.com/lavaorg/lrt/mlog"
	"github.com/lavaorg/warp/osrv"
	"github.com/lavaorg/warp/warp9"
)

var addr = flag.String("a", ":5640", "network address")
var debug = flag.Int("debug", 0, "print debug messages")
var oserver = flag.String("s", "echoos", "object server")

type w9os struct {
	warp9.Srv
	warp9.StatsOps
}

var root *osrv.WODir

func main() {
	flag.Parse()
	oserv := getSrv()
	setup()
	err := oserv.StartNetListener("tcp", *addr)
	if err != nil {
		log.Println(err)
	}
}

func getSrv() *w9os {
	srv := new(w9os)

	srv.Id = "w9"
	srv.Debuglevel = *debug
	srv.Start(srv)
	fmt.Print("nullfs starting\n")
	return srv
}

func setup() {

	nullfs.Setup()
	// setup a root
	srv := osrv.Servers[*oserver]
	if srv == nil {
		panic("no server: " + *oserver)
	}
	root = srv.RootDir()
}

func (ufs *w9os) Attach(req *warp9.SrvReq) {
	if req.Afid != nil {
		req.RespondError(warp9.Enoauth)
		return
	}
	//tc := req.Tc
	// ignore the aname; just mount "/"
	//rm fid := new(nullfsFid)
	//rm fid.entry = root
	req.Fid.Aux = root
	req.RespondRattach(&root.Qid)
}

func (*w9os) Flush(req *warp9.SrvReq) {}

func (*w9os) Walk(req *warp9.SrvReq) {
	wo := req.Fid.Aux.(*osrv.WODir)
	tc := req.Tc

	if wo == nil {
		req.RespondError(warp9.Ebaduse)
		return
	}

	//if req.Newfid.Aux == nil {
	//	req.Newfid.Aux = new(nullfsFid)
	//}

	// there are no entries so if path is not "." or ".." or "/" return an error
	// "." and ".." by definition are alias for the current node, so valid.
	if len(tc.Wname) != 1 {
		req.RespondError(warp9.Enotexist) //warp9.Enoent)
		return
	}
	p := tc.Wname[0]
	if p != "." && p != ".." && p != "/" {
		req.RespondError(warp9.Enotexist)
		return
	}

	req.Newfid.Aux = req.Fid.Aux
	d := wo.DirEntry()

	req.RespondRwalk(&d.Qid)
}

func (*w9os) Open(req *warp9.SrvReq) {

	tc := req.Tc
	mode := tc.Mode
	if mode != warp9.OREAD {
		req.RespondError(warp9.Eperm)
		return
	}

	req.RespondRopen(&root.Qid, 0)
}

func (*w9os) Create(req *warp9.SrvReq) {
	// no creation
	req.RespondError(warp9.Enotimpl)
}

func (*w9os) Read(req *warp9.SrvReq) {
	tc := req.Tc
	rc := req.Rc

	rc.InitRread(tc.Count)

	// convert our directory to byte buffer; we aren't caching
	b := warp9.PackDir(&root.Dir)

	// determine which and how many bytes to return
	var count int
	switch {
	case tc.Offset > uint64(len(b)):
		count = 0
	case len(b[tc.Offset:]) > int(tc.Count):
		count = int(tc.Count)
	default:
		count = len(b[tc.Offset:])
	}
	copy(rc.Data, b[tc.Offset:int(tc.Offset)+count])
	log.Printf("buf:%v, rc.Data: %v, off:%v,  count:%v\n", len(b), len(rc.Data), tc.Offset, count)
	rc.SetRreadCount(uint32(count))
	req.Respond()
}

func (*w9os) Write(req *warp9.SrvReq) {
	req.RespondError(warp9.Enotimpl)
	return
}

func (*w9os) Clunk(req *warp9.SrvReq) { req.RespondRclunk() }

func (*w9os) Remove(req *warp9.SrvReq) {
	req.RespondError(warp9.Enotimpl)
	return
}

func (*w9os) Stat(req *warp9.SrvReq) {
	wo := req.Fid.Aux.(*osrv.WODir)
	if wo == nil {
		req.RespondError(warp9.Ebaduse)
	}
	req.RespondRstat(wo.DirEntry())
	return
}
func (u *w9os) Wstat(req *warp9.SrvReq) {
	req.RespondError(warp9.Enotimpl)
	return
}
