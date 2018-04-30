// Copyright 2018 Larry Rau. All rights reserved
// See Apache2 LICENSE

package w9

import (
	"time"

	"github.com/lavaorg/lrt/mlog"
	"github.com/lavaorg/warp/warp9"
)

// item represents a generic node in a hierarchical tree.  This interface
// will allow the object server to perform generic operations on any node.
// Where necessary it can learn more details of what the node is (e.g. a directory,
// or a bind-point, etc)
type Item interface {
	GetDir() *warp9.Dir
	GetItem() Item
	Data() []byte
	IsDirectory() Directory
	Opened() bool
	SetOpened(b bool) bool
	Parent() Directory
	SetParent(d Directory) error
	Read(obuf []byte, off uint64, rcount uint32) (uint32, error)
}

type Directory interface {
	Create(name string, perm uint32, mode uint8) (Item, error)
	Walk(path []string) (Item, error)
}

// OneItem is a holder for an object in a tree.
// It implements the Item interface
type OneItem struct {
	warp9.Dir
	parent Directory
	opened bool
	Buffer []byte
}

type DirItem struct {
	OneItem
	Content map[string]Item
}

func (o *OneItem) GetDir() *warp9.Dir {
	return &o.Dir
}

func (o *OneItem) GetItem() Item {
	return o
}

func (o *OneItem) Data() []byte {
	return o.Buffer
}

func (o *OneItem) IsDirectory() Directory {
	return nil
}

// set to opened; return previous status
func (o *OneItem) SetOpened(v bool) bool {
	b := o.opened
	o.opened = v
	return b
}

func (o *OneItem) Opened() bool {
	return o.opened
}

func (o *OneItem) Parent() Directory {
	return o.parent
}

func (o *OneItem) SetParent(d Directory) error {
	o.parent = d
	return nil
}

func (o *OneItem) Read(obuf []byte, off uint64, rcount uint32) (uint32, error) {

	// determine which and how many bytes to return
	var count uint32
	switch {
	case off > uint64(len(o.Buffer)):
		count = 0
	case uint32(len(o.Buffer[off:])) > rcount:
		count = rcount
	default:
		count = uint32(len(o.Buffer[off:]))
	}
	copy(obuf, o.Buffer[off:uint32(off)+count])
	mlog.Debug("d.BUffer:%v, obuf: %v, off:%v, rcount:%v\n", len(o.Buffer), len(obuf), off, count)

	return count, nil

}

//
// create a new empty directory
//
func NewDirItem() *DirItem {
	d := new(DirItem)
	d.Content = make(map[string]Item, 0)
	return d
}

// return the current buffer. Do not modify.
func (d *DirItem) GetData() []byte {
	return nil
}

func (d *DirItem) IsDirectory() Directory {
	return d
}

func (d *DirItem) Create(name string, perm uint32, mode uint8) (Item, error) {

	var i Item
	nqid := warp9.Qid{warp9.QTOBJ, 0, NextQid()}

	if perm&warp9.DMDIR > 0 {
		i = NewDirItem()
		nqid.Type = warp9.QTDIR
	} else {
		i = new(OneItem)
	}
	i.SetParent(d)
	ndir := i.GetDir()
	ndir.Name = name
	ndir.Qid = nqid
	ndir.Uid = d.Uid
	ndir.Gid = d.Gid
	ndir.Muid = ndir.Uid

	ndir.Atime = uint32(time.Now().Unix())
	ndir.Mtime = d.Atime

	ndir.Mode = perm

	d.Content[name] = i
	return i, nil
}

//
// walk the content items looking for a match for path. each element in path
// except the last must be a directory.
// return the found Item or an error
func (d *DirItem) Walk(path []string) (Item, error) {

	if len(path) < 1 {
		// empty path succeeds in finding self
		return d, nil
	}

	if len(path) == 1 {
		// leaf item return if found
		n := path[0]
		var item Item
		if n == ".." {
			item = d.parent.(Item)
		} else {
			item = d.Content[n]
		}
		if item == nil {
			return nil, warp9.Enotexist
		}
		return item, nil
	}

	// element must be a diretory to further walk
	elem := path[0]
	path = path[1:]
	var item Item
	var dir Directory
	if elem == ".." {
		dir = d.Parent()
	} else {
		item := d.Content[elem]
		if item == nil {
			return nil, warp9.Enotexist
		}
		dir = item.IsDirectory()
	}
	if dir == nil {
		return item, warp9.Enotdir
	}
	// walk to next dir
	return dir.Walk(path)
}

func (d *DirItem) SetOpened(o bool) bool {
	b := d.opened
	d.opened = o
	if !o {
		d.Buffer = nil
	}
	return b
}

func (d *DirItem) Read(obuf []byte, off uint64, rcount uint32) (uint32, error) {

	// walk all contents; get Dir structure; pack as bytes
	if d.Buffer == nil {
		d.Buffer = make([]byte, 0, 300)
		for _, item := range d.Content {
			buf := warp9.PackDir(item.GetDir())
			d.Buffer = append(d.Buffer, buf...)
			mlog.Debug("dir item:%v, len(buf):%v, len(Buffer):%v", item, len(buf), len(d.Buffer))
		}
	}

	// determine which and how many bytes to return
	var count uint32
	switch {
	case off > uint64(len(d.Buffer)):
		count = 0
	case uint32(len(d.Buffer[off:])) > rcount:
		count = rcount
	default:
		count = uint32(len(d.Buffer[off:]))
	}
	copy(obuf, d.Buffer[off:uint32(off)+count])
	mlog.Debug("d.BUffer:%v, obuf: %v, off:%v, rcount:%v\n", len(d.Buffer), len(obuf), off, count)

	return count, nil
}
