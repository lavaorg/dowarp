// Copyright 2018 Larry Rau. All rights reserved
// See Apache2 LICENSE

package w9

import (
	_ "github.com/lavaorg/warp/warp9"
)

type Command struct {
	OneItem
	Buf     string
	Results []byte
	fcts    map[string]CommandFct
	Ctx     interface{}
}

type CommandFct func(cmd *Command, cmdname string) error

func NewCommand(name string, fcts map[string]CommandFct) *Command {
	var cmd Command

	cmd.Name = name

	cmd.fcts = fcts
	cmd.Results = make([]byte, 20, 80)

	return &cmd
}
