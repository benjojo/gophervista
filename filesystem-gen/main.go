package main

import (
	"database/sql"
	"encoding/base32"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbpath := flag.String("dbpath", "./crawl.db", "Where to find the database")
	opath := flag.String("outputpath", "./fat32/", "Where to find the database")
	flag.Parse()
	log.Printf("Making filesystem...")

	db, err := sql.Open("sqlite3", *dbpath)
	if err != nil {
		log.Fatalf("unable to open db %s", err.Error())
	}

	err = db.Ping() // database do odd things
	if err != nil {
		log.Fatalf("unable to check db %s", err.Error())
	}

	filesmoved := 0
	os.Mkdir(*opath+"0/", 0777)

	filepath.Walk("./data", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		encodedfilename := filepath.Base(path)
		gopherurl, err := base32.StdEncoding.DecodeString(encodedfilename)
		if err != nil {
			return nil
		}

		var dbid int
		err = db.QueryRow("SELECT id FROM assets WHERE path=? LIMIT 1", string(gopherurl)).Scan(&dbid)

		if err != nil || dbid == 0 {
			log.Printf("%s", err.Error())
			return nil
		}

		if filesmoved%1000 == 0 {
			os.Mkdir(fmt.Sprintf("%s%d/", *opath, filesmoved/1000), 0777)
		}

		dstpath := fmt.Sprintf("%s%d/%d.txt", *opath, filesmoved/1000, dbid)
		err = copy(path, dstpath)
		if err != nil {
			log.Printf("failed to copy file over to %s - %s", dstpath, err.Error())
		}

		filesmoved++

		return nil
	})
}

func copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}
