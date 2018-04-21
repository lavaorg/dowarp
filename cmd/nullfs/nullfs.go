// Copyright 2018 Larry Rau. All rights reserved
// See Apache2 LICENSE

package nullfs

/*
import (
	"flag"
	"fmt"
	"log"

	"github.com/lavaorg/dowarp/nullfs"
	"github.com/lavaorg/warp/warp9"
)

var addr = flag.String("addr", ":5640", "network address")
var debug = flag.Int("debug", 0, "print debug messages")

func main() {
	flag.Parse()
	nullfs := new(nullfs.Nullfs)
	showInterfaces(nullfs)

	nullfs.Id = "nullfs"
	nullfs.Debuglevel = *debug
	nullfs.Start(nullfs)
	fmt.Print("nullfs starting\n")

	err := nullfs.StartNetListener("tcp", *addr)
	if err != nil {
		log.Println(err)
	}
}

func showInterfaces(ifaces interface{}) {
	if _, ok := (ifaces).(warp9.SrvReqOps); ok {
		fmt.Println("implements: SrvReqOps")
	}
	if _, ok := (ifaces).(warp9.StatsOps); ok {
		fmt.Println("implements: StatsOps")
	}

}
*/
