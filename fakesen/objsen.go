// Copyright 2018 Larry Rau. All rights reserved
// See Apache2 LICENSE

/*
simulated fake sensors
*/
package fakesen

import (
	"fmt"

	"github.com/lavaorg/lrt/mlog"
	"github.com/lavaorg/warp/warp9"
	"github.com/lavaorg/warp/wkit"
)

type FakeSensor struct {
	wkit.OneItem
	lasttemp float32
}

func NewFakeSensor() *FakeSensor {
	var s FakeSensor
	s.lasttemp = 32.8
	return &s
}

func (o *FakeSensor) Walked() (wkit.Item, error) {
	mlog.Debug("Fake...Walked:%T.%v", o, o)
	return o, nil
}

// Return the requested set of bytes from the object's byte buffer.
// ensure rcount is >= sizeof(sensor-reading-txt); else return error
func (o *FakeSensor) Read(obuf []byte, off uint64, rcount uint32) (uint32, error) {
	if o.lasttemp > 50 {
		o.lasttemp = 28
	}
	o.lasttemp += 5
	b := []byte(fmt.Sprintf("%f", o.lasttemp))
	if off > uint64(len(b)) {
		return 0, nil
	}
	if int(rcount) < len(b) {
		return 0, warp9.Eio
	}
	n := copy(obuf, b)
	return uint32(n), nil
}
