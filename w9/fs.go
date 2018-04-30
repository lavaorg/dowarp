// Copyright 2018 Larry Rau. All rights reserved
// See Apache2 LICENSE

package w9

import (
	"github.com/lavaorg/lrt/mlog"
	"github.com/lavaorg/warp/warp9"
)

func (srv *Serv) Attach(req *warp9.SrvReq) {
	if req.Afid != nil {
		req.RespondError(warp9.Enoauth)
		return
	}
	//tc := req.Tc
	// ignore the aname; just mount "/"
	//rm fid := new(nullfsFid)
	//rm fid.entry = root
	req.Fid.Aux = srv.root
	mlog.Info("req.Fid:%v, root:%v", req.Fid, srv.root)
	req.RespondRattach(&srv.root.Qid)
}

func (*Serv) Flush(req *warp9.SrvReq) {}

func (*Serv) Walk(req *warp9.SrvReq) {
	d, ok := req.Fid.Aux.(Directory)
	if !ok {
		req.RespondError(warp9.Enotdir)
		return
	}
	if d == nil {
		req.RespondError(warp9.Ebaduse)
		return
	}

	tc := req.Tc

	item, err := d.Walk(tc.Wname)
	if err != nil {
		req.RespondError(fsRespondError(err, warp9.Enoent))
		return
	}
	req.Newfid.Aux = item
	req.RespondRwalk(&item.GetDir().Qid)
}

func (*Serv) Open(req *warp9.SrvReq) {
	i := req.Fid.Aux.(Item)
	tc := req.Tc
	mode := tc.Mode
	if mode != warp9.OREAD {
		req.RespondError(warp9.Eperm)
		return
	}

	// check permissions

	i.SetOpened(true)

	req.RespondRopen(&i.GetDir().Qid, 0)
}

func (*Serv) Clunk(req *warp9.SrvReq) {
	i := req.Fid.Aux.(Item)
	i.SetOpened(false)
	req.RespondRclunk()
}

func (*Serv) Create(req *warp9.SrvReq) {
	d, ok := req.Fid.Aux.(Directory)
	if !ok {
		req.RespondError(warp9.Enotdir)
	}
	if d == nil {
		req.RespondError(warp9.Ebaduse)
		return
	}

	tc := req.Tc

	item, err := d.Create(tc.Name, tc.Perm, tc.Mode)
	if err != nil {
		req.RespondError(fsRespondError(err, warp9.Eio))
		return
	}

	req.Fid.Aux = item
	req.RespondRcreate(&item.GetDir().Qid, 0)
}

func (*Serv) Read(req *warp9.SrvReq) {
	item := req.Fid.Aux.(Item)
	tc := req.Tc
	rc := req.Rc

	rc.InitRread(tc.Count)

	count, err := item.Read(rc.Data, tc.Offset, tc.Count)
	if err != nil {
		req.RespondError(fsRespondError(err, warp9.Eio))
		return
	}
	rc.SetRreadCount(count)
	req.Respond()
}

func (*Serv) Write(req *warp9.SrvReq) {
	req.RespondError(warp9.Enotimpl)
	return
}

func (*Serv) Remove(req *warp9.SrvReq) {
	req.RespondError(warp9.Enotimpl)
	return
}

func (*Serv) Stat(req *warp9.SrvReq) {
	wo := req.Fid.Aux.(Item)
	if wo == nil {
		req.RespondError(warp9.Ebaduse)
	}
	req.RespondRstat(wo.GetDir())
	return
}

func (u *Serv) Wstat(req *warp9.SrvReq) {
	req.RespondError(warp9.Enotimpl)
	return
}

// helper functions

func fsRespondError(err error, alterr warp9.W9Err) warp9.W9Err {
	err9, ok := err.(warp9.W9Err)
	if !ok {
		err9 = alterr
	}
	return err9
}
