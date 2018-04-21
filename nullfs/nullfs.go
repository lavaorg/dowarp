// Copyright 2018 Larry Rau. All rights reserved
// See Apache2 LICENSE

// Nullfs is primarily a template to start building multi-level FS from.
// This is a read only file system with one file /info.  This means you can
// open/read/stat/ the files "/" and "/info"
// "/" is a directory
// "/info" is a read only file
//
package nullfs

import (
	"log"
	"time"

	"github.com/lavaorg/lrt/mlog"
	"github.com/lavaorg/warp/osrv"
	"github.com/lavaorg/warp/warp9"
)

type nullfsFid struct {
	entry       *osrv.WODir
	direntrybuf []byte
}

type NullOSrv struct {
}

// NullDir represents an entry in the NullFS. It will contain a ninep dir
// but can carry additional data/state needed as necessary
type NullDir osrv.WODir

var nullosrv *NullOSrv = &(NullOSrv{}) //our server
var root *osrv.WODir = newNullDir("/") //our root directory

func Setup( /*mt osrv.Mount*/ ) {
	osrv.Register(nullosrv)
	mlog.Info("nullos registered")

	osrv.Handle("/")
}

func newNullDir(n string) *osrv.WODir {
	var d osrv.WODir

	d.Name = n
	d.Uid = 501
	d.Gid = 20
	d.Muid = 501

	d.Mode = warp9.DMDIR | uint32(perms(warp9.DMREAD, warp9.DMREAD, warp9.DMREAD))
	d.Atime = uint32(time.Now().Unix())
	d.Mtime = d.Atime

	d.Qid = warp9.Qid{warp9.QTDIR, 0, 9999}

	return &d
}

func (nos *NullOSrv) Name() string {
	return "nullosrv"
}

func (nos *NullOSrv) RootDir() *osrv.WODir {
	return root
}

func perms(u, g, o byte) uint16 {
	return uint16(uint16(u)<<6 | uint16(g)<<3 | uint16(o))
}

func (*NullOSrv) FidDestroy(sfid *warp9.SrvFid) {
	if sfid.Aux == nil {
		return
	}

	fid := sfid.Aux.(osrv.WOBase)
	if sfid.Fconn.Debuglevel > 0 {
		log.Printf("fid destroy:%v\n", fid)
	}
	//cleanup fid
}
