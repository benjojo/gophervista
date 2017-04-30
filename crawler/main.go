package main

import (
	"flag"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var (
	databasepath = flag.String("dbpath", "./crawl.db", "Where to store the metadata of crawling")
	datapath     = flag.String("datadir", "./data", "where to download assets to")
)

func main() {
	flag.Parse()

	ledger := crawlerQueue{}
	err := ledger.Init(*databasepath)
	if err != nil {
		log.Fatalf("Unable to open database. %s", err.Error())
	}

	crawl(ledger, *datapath)
	os.Exit(1)
}
