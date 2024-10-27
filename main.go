package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/im-karina/basics/cfg"
	"github.com/im-karina/basics/db"
	"github.com/im-karina/basics/srv"
)

func main() {
	flag.Parse()
	cfg.Load()

	taskFns = make(map[string]func(s string) error)

	taskFns["db:drop"] = db.Drop
	taskFns["db:migrate"] = db.Migrate
	taskFns["db:rollback"] = db.Rollback
	taskFns["db:schema:dump"] = db.DumpSchema
	taskFns["db:wal_cleanup"] = db.WalCleanup
	taskFns["serve"] = srv.Serve

	log.Println("connecting to database")
	db.MustConnectOnce()
	defer db.Db.Close()

	for _, cmd := range flag.Args() {
		DoTask(cmd)
	}
}

var taskFns map[string]func(s string) error

func DoTask(s string) {
	fn, ok := taskFns[s]
	if !ok {
		log.Fatalln("unknown task:", s)
	}
	fmt.Println(">", s)
	fn(s)
	fmt.Println("done")
}
