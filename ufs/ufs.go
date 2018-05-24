// Copyright 2009 The go9p Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ufs serves up a designted portion of the host file system.
// This fs is primairly for tools or testing and not meant for produciton.
package ufs

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lavaorg/warp/warp9"
)

type ufsFid struct {
	path       string
	file       *os.File
	dirs       []os.FileInfo
	direntends []int
	dirents    []byte
	diroffset  uint64
	st         os.FileInfo
}

type Ufs struct {
	warp9.Srv
	warp9.StatsOps
	Root string
}

func toError(err error) *warp9.WarpError {
	// could convert err string to an id?
	return warp9.ErrorMsg(warp9.Eio, err.Error())
}

// Error codes for UFS
const (
	EUFSstat = iota + 900
	EUFSopen
	EUFScreate
	EUFSread
	EUFSwrite
	EUFSremove
	EUFSchmod
	EUFSchown
	EUFSrename
	EUFStruncate
)

// IsBlock reports if the file is a block device
func isBlock(d os.FileInfo) bool {
	stat := d.Sys().(*syscall.Stat_t)
	return (stat.Mode & syscall.S_IFMT) == syscall.S_IFBLK
}

// IsChar reports if the file is a character device
func isChar(d os.FileInfo) bool {
	stat := d.Sys().(*syscall.Stat_t)
	return (stat.Mode & syscall.S_IFMT) == syscall.S_IFCHR
}

func (fid *ufsFid) stat() *warp9.WarpError {
	var err error
	fid.st, err = os.Lstat(fid.path)
	if err != nil {
		return warp9.Error(EUFSstat)
	}
	return nil
}

func omode2uflags(mode uint8) int {
	ret := int(0)
	switch mode & 3 {
	case warp9.OREAD:
		ret = os.O_RDONLY
		break

	case warp9.ORDWR:
		ret = os.O_RDWR
		break

	case warp9.OWRITE:
		ret = os.O_WRONLY
		break

	case warp9.OUSE:
		ret = os.O_RDONLY
		break
	}

	if mode&warp9.OTRUNC != 0 {
		ret |= os.O_TRUNC
	}

	return ret
}

func dir2Qid(d os.FileInfo) *warp9.Qid {
	var qid warp9.Qid

	qid.Path = d.Sys().(*syscall.Stat_t).Ino
	qid.Version = uint32(d.ModTime().UnixNano() / 1000000)
	qid.Type = dir2QidType(d)

	return &qid
}

func dir2QidType(d os.FileInfo) uint8 {
	ret := uint8(0)
	if d.IsDir() {
		ret |= warp9.QTDIR
	}
	return ret
}

func dir2Npmode(d os.FileInfo) uint32 {
	ret := uint32(d.Mode() & 0777)
	if d.IsDir() {
		ret |= warp9.DMDIR
	}
	return ret
}

// Dir is an instantiation of the warp9.Dir structure
// that can act as a receiver for local methods.
type ufsDir struct {
	warp9.Dir
}

func dir2Dir(path string, d os.FileInfo, upool warp9.Users) (*warp9.Dir, error) {
	if r := recover(); r != nil {
		fmt.Print("stat failed: ", r)
		return nil, &os.PathError{"dir2Dir", path, nil}
	}
	sysif := d.Sys()
	if sysif == nil {
		return nil, &os.PathError{"dir2Dir: sysif is nil", path, nil}
	}
	sysMode := sysif.(*syscall.Stat_t)

	dir := new(ufsDir)
	dir.Qid = *dir2Qid(d)
	dir.Mode = dir2Npmode(d)
	dir.Atime = 0 //uint32(sysMode.Atim)
	dir.Mtime = uint32(d.ModTime().Unix())
	dir.Length = uint64(d.Size())
	dir.Name = path[strings.LastIndex(path, "/")+1:]

	dir.Uid = sysMode.Uid
	dir.Gid = sysMode.Gid

	return &dir.Dir, nil
}

func (*Ufs) ConnOpened(conn *warp9.Conn) {
	if conn.Srv.Debuglevel > 0 {
		log.Println("connected")
	}
}

func (*Ufs) ConnClosed(conn *warp9.Conn) {
	if conn.Srv.Debuglevel > 0 {
		log.Println("disconnected")
	}
}

func (*Ufs) FidDestroy(sfid *warp9.SrvFid) {
	var fid *ufsFid

	if sfid.Aux == nil {
		return
	}

	fid = sfid.Aux.(*ufsFid)
	if fid.file != nil {
		fid.file.Close()
	}
}

func (ufs *Ufs) Attach(req *warp9.SrvReq) {
	if req.Afid != nil {
		req.RespondError(warp9.Error(warp9.Enoauth))
		return
	}

	tc := req.Tc
	fid := new(ufsFid)
	// You can think of the ufs.Root as a 'chroot' of a sort.
	// clients attach are not allowed to go outside the
	// directory represented by ufs.Root
	fid.path = path.Join(ufs.Root, tc.Aname)

	req.Fid.Aux = fid
	err := fid.stat()
	if err != nil {
		req.RespondError(err)
		return
	}

	qid := dir2Qid(fid.st)
	req.RespondRattach(qid)
}

func (*Ufs) Flush(req *warp9.SrvReq) {}

func (*Ufs) Walk(req *warp9.SrvReq) {
	fid := req.Fid.Aux.(*ufsFid)
	tc := req.Tc

	err := fid.stat()
	if err != nil {
		req.RespondError(err)
		return
	}

	if req.Newfid.Aux == nil {
		req.Newfid.Aux = new(ufsFid)
	}

	nfid := req.Newfid.Aux.(*ufsFid)
	//wqids := make([]warp9.Qid, len(tc.Wname))
	path := fid.path
	i := 0
	p := path
	for ; i < len(tc.Wname); i++ {
		p = p + "/" + tc.Wname[i]
	}
	st, e := os.Lstat(p)
	if e != nil {
		req.RespondError(warp9.Error(warp9.Enotexist))
		return
	}

	wqid := *dir2Qid(st)
	path = p

	nfid.path = path
	req.RespondRwalk(&wqid)
}

func (*Ufs) Open(req *warp9.SrvReq) {
	fid := req.Fid.Aux.(*ufsFid)
	tc := req.Tc
	err := fid.stat()
	if err != nil {
		req.RespondError(err)
		return
	}

	var e error
	fid.file, e = os.OpenFile(fid.path, omode2uflags(tc.Mode), 0)
	if e != nil {
		req.RespondError(warp9.Error(EUFSopen))
		return
	}

	req.RespondRopen(dir2Qid(fid.st), 0)
}

func (*Ufs) Create(req *warp9.SrvReq) {
	fid := req.Fid.Aux.(*ufsFid)
	tc := req.Tc
	err := fid.stat()
	if err != nil {
		req.RespondError(err)
		return
	}

	path := fid.path + "/" + tc.Name
	var e error = nil
	var file *os.File = nil
	switch {
	case tc.Perm&warp9.DMDIR != 0:
		e = os.Mkdir(path, os.FileMode(tc.Perm&0777))

	default:
		var mode uint32 = tc.Perm & 0777
		file, e = os.OpenFile(path, omode2uflags(tc.Mode)|os.O_CREATE, os.FileMode(mode))
	}

	if file == nil && e == nil {
		file, e = os.OpenFile(path, omode2uflags(tc.Mode), 0)
	}

	if e != nil {
		req.RespondError(warp9.Error(EUFSopen))
		return
	}

	fid.path = path
	fid.file = file
	err = fid.stat()
	if err != nil {
		req.RespondError(err)
		return
	}

	req.RespondRcreate(dir2Qid(fid.st), 0)
}

func (*Ufs) Read(req *warp9.SrvReq) {
	fid := req.Fid.Aux.(*ufsFid)
	tc := req.Tc
	rc := req.Rc
	err := fid.stat()
	if err != nil {
		req.RespondError(err)
		return
	}

	rc.InitRread(tc.Count)
	var count int
	var e error
	if fid.st.IsDir() {
		if tc.Offset == 0 {
			var e error
			// If we got here, it was open. Can't really seek
			// in most cases, just close and reopen it.
			fid.file.Close()
			if fid.file, e = os.OpenFile(fid.path, omode2uflags(req.Fid.Omode), 0); e != nil {
				req.RespondError(warp9.Error(EUFSopen))
				return
			}

			if fid.dirs, e = fid.file.Readdir(-1); e != nil {
				req.RespondError(warp9.Error(EUFSread))
				return
			}

			fid.dirents = nil
			fid.direntends = nil
			for i := 0; i < len(fid.dirs); i++ {
				path := fid.path + "/" + fid.dirs[i].Name()
				st, _ := dir2Dir(path, fid.dirs[i], req.Conn.Srv.Upool)
				if st == nil {
					continue
				}
				b := warp9.PackDir(st)
				fid.dirents = append(fid.dirents, b...)
				count += len(b)
				fid.direntends = append(fid.direntends, count)
			}
		}

		switch {
		case tc.Offset > uint64(len(fid.dirents)):
			count = 0
		case len(fid.dirents[tc.Offset:]) > int(tc.Count):
			count = int(tc.Count)
		default:
			count = len(fid.dirents[tc.Offset:])
		}

		copy(rc.Data, fid.dirents[tc.Offset:int(tc.Offset)+count])

	} else {
		count, e = fid.file.ReadAt(rc.Data, int64(tc.Offset))
		if e != nil && e != io.EOF {
			req.RespondError(warp9.Error(EUFSread))
			return
		}

	}

	rc.SetRreadCount(uint32(count))
	req.Respond()
}

func (*Ufs) Write(req *warp9.SrvReq) {
	fid := req.Fid.Aux.(*ufsFid)
	tc := req.Tc
	err := fid.stat()
	if err != nil {
		req.RespondError(err)
		return
	}

	n, e := fid.file.WriteAt(tc.Data, int64(tc.Offset))
	if e != nil {
		req.RespondError(warp9.Error(EUFSwrite))
		return
	}

	req.RespondRwrite(uint32(n))
}

func (*Ufs) Clunk(req *warp9.SrvReq) { req.RespondRclunk() }

func (*Ufs) Remove(req *warp9.SrvReq) {
	fid := req.Fid.Aux.(*ufsFid)
	err := fid.stat()
	if err != nil {
		req.RespondError(err)
		return
	}

	e := os.Remove(fid.path)
	if e != nil {
		req.RespondError(warp9.Error(EUFSremove))
		return
	}

	req.RespondRremove()
}

func (*Ufs) Stat(req *warp9.SrvReq) {
	fid := req.Fid.Aux.(*ufsFid)
	err := fid.stat()
	if err != nil {
		req.RespondError(err)
		return
	}

	st, _ := dir2Dir(fid.path, fid.st, req.Conn.Srv.Upool)
	if st == nil {
		req.RespondError(warp9.Error(EUFSstat))
		return
	}

	req.RespondRstat(st)
}

func lookup(uid string, group bool) (uint32, *warp9.WarpError) {
	if uid == "" {
		return warp9.NOUID, nil
	}
	usr, e := user.Lookup(uid)
	if e != nil {
		return warp9.NOUID, warp9.Error(warp9.Ebaduser)
	}
	conv := usr.Uid
	if group {
		conv = usr.Gid
	}
	u, e := strconv.Atoi(conv)
	if e != nil {
		return warp9.NOUID, warp9.Error(warp9.Ebaduser)
	}
	return uint32(u), nil
}

func (u *Ufs) Wstat(req *warp9.SrvReq) {
	fid := req.Fid.Aux.(*ufsFid)
	err := fid.stat()
	if err != nil {
		req.RespondError(err)
		return
	}

	dir := &req.Tc.Dir
	if dir.Mode != 0xFFFFFFFF {
		mode := dir.Mode & 0777
		e := os.Chmod(fid.path, os.FileMode(mode))
		if e != nil {
			req.RespondError(warp9.Error(EUFSchmod))
			return
		}
	}

	uid := dir.Uid
	gid := dir.Gid

	if uid != warp9.NOUID || gid != warp9.NOUID {
		e := os.Chown(fid.path, int(uid), int(gid))
		if e != nil {
			req.RespondError(warp9.Error(EUFSchown))
			return
		}
	}

	if dir.Name != "" {
		fmt.Printf("Rename %s to %s\n", fid.path, dir.Name)
		// if first char is / it is relative to root, else relative to
		// cwd.
		var destpath string
		if dir.Name[0] == '/' {
			destpath = path.Join(u.Root, dir.Name)
			fmt.Printf("/ results in %s\n", destpath)
		} else {
			fiddir, _ := path.Split(fid.path)
			destpath = path.Join(fiddir, dir.Name)
			fmt.Printf("rel  results in %s\n", destpath)
		}
		err := syscall.Rename(fid.path, destpath)
		fmt.Printf("rename %s to %s gets %v\n", fid.path, destpath, err)
		if err != nil {
			req.RespondError(warp9.Error(EUFSrename))
			return
		}
		fid.path = destpath
	}

	if dir.Length != 0xFFFFFFFFFFFFFFFF {
		e := os.Truncate(fid.path, int64(dir.Length))
		if e != nil {
			req.RespondError(warp9.Error(EUFStruncate))
			return
		}
	}

	// If either mtime or atime need to be changed, then
	// we must change both.
	if dir.Mtime != ^uint32(0) || dir.Atime != ^uint32(0) {
		mt, at := time.Unix(int64(dir.Mtime), 0), time.Unix(int64(dir.Atime), 0)
		if cmt, cat := (dir.Mtime == ^uint32(0)), (dir.Atime == ^uint32(0)); cmt || cat {
			st, e := os.Stat(fid.path)
			if e != nil {
				req.RespondError(warp9.Error(EUFSstat))
				return
			}
			switch cmt {
			case true:
				mt = st.ModTime()
			default:
				//at = time.Time(0)//atime(st.Sys().(*syscall.Stat_t))
			}
		}
		e := os.Chtimes(fid.path, at, mt)
		if e != nil {
			req.RespondError(warp9.Error(EUFSstat))
			return
		}
	}

	req.RespondRwstat()
}
