package main

import (
	"github.com/urfave/cli"
	"os"
	"syscall"
	"os/signal"
	"time"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/texas"
)

const (
	TableCountFName = "t_count"
	TableSeatCountFName = "ts_count"
	TableLevelFName = "t_level"
	PortFName = "port"
)

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag {
		cli.IntFlag{ Name: TableCountFName, Value: 10 },
		cli.IntFlag{ Name: TableSeatCountFName, Value: 5 },
		cli.IntFlag{ Name: TableLevelFName, Value: 1 },
		cli.IntFlag{ Name: PortFName, Value: 3030 },
	}
	app.Action = run

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func run(c *cli.Context) {
	room := texas.NewRoomServer(c.Int(TableCountFName), c.Int(TableSeatCountFName), c.Int(TableLevelFName), c.Int(PortFName))
	if err := room.Start(); err != nil {
		panic(err)
	}
	signalListen(func() {
		if err := room.Stop(); err != nil {
			panic(err)
		}
		time.Sleep(1 * time.Second)
	})
}

// listen stop signal
func signalListen(stopFunc func()) {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	<-c

	stopFunc()
}