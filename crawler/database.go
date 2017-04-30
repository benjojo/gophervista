package main

import (
	"database/sql"
	"log"

	"time"

	"regexp"

	_ "github.com/mattn/go-sqlite3"
)

type crawlerQueue struct {
	db *sql.DB
}

func (cq *crawlerQueue) Init(databasepath string) error {
	db, err := sql.Open("sqlite3", databasepath)
	if err != nil {
		return err
	}

	err = db.Ping() // database do odd things
	if err != nil {
		return err
	}

	cq.db = db

	cq.maketableifnotexists("assets",
		"CREATE TABLE assets (id INTEGER PRIMARY KEY, timestamp NUMERIC, path TEXT, type TEXT, lastcrawled NUMERIC);")

	cq.maketableifnotexists("restrictions",
		"CREATE TABLE `restrictions` (`id` INTEGER,`pattern` TEXT,`nolearn` INTEGER DEFAULT 0,`nocrawl` INTEGER DEFAULT 0,PRIMARY KEY(id));")

	return nil
}

func (cq *crawlerQueue) maketableifnotexists(table, makestring string) {
	// Check that the table for radio info exists
	tablename := ""
	cq.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", table).Scan(&tablename)

	if tablename != table {
		log.Printf("Made a new %s table", table)
		cq.db.Exec(makestring)
	}

}

func (cq *crawlerQueue) GetItemToCrawl() string {
	asset := ""
	err := cq.db.QueryRow("SELECT path FROM assets WHERE lastcrawled < ? LIMIT 1", time.Now().Unix()-604800).Scan(&asset)
	if err != nil {
		return ""
	}
	return asset
}

func (cq *crawlerQueue) FlagItemAsCrawled(path string) {
	_, err := cq.db.Exec("UPDATE assets SET lastcrawled = ? WHERE path = ?", time.Now().Unix(), path)
	if err != nil {
		log.Printf("Unable to flag asset as crawled? %s", err.Error())
	}
}

func (cq *crawlerQueue) FlagItem(path string, gtypestr string) {
	test := 0
	err := cq.db.QueryRow("SELECT COUNT(*) path FROM assets WHERE path = ?;", path).Scan(&test)
	if err != nil {
		log.Printf("Huh, unable to see if an item to be should be crawled -  %s", err.Error())
		return
	}

	if test == 0 {
		// Fantastic, We can add it
		_, err = cq.db.Exec("INSERT INTO `assets` (`id`,`timestamp`,`path`,`type`,`lastcrawled`) VALUES (NULL,?,?,?,0);", time.Now().Unix(), path, gtypestr)
		if err != nil {
			log.Printf("Huh, unable to flag to be should be crawled -  %s", err.Error())
			return
		}
	}
}

type permissionRow struct {
	Pattern  *regexp.Regexp
	CanLearn bool
	CanCrawl bool
}

var permissionsCache []permissionRow

func (cq *crawlerQueue) CheckItemPerms(path string) (canCrawl bool, canLearn bool) {
	if len(permissionsCache) == 0 || permissionsCache == nil {
		log.Printf("Loading perms")
		permissionsCache = make([]permissionRow, 0)
		rows, err := cq.db.Query("SELECT pattern,nolearn,nocrawl FROM restrictions;")
		if err != nil {
			log.Fatalf("Unable to load crawling restrictions/perms, %s", err.Error())
		}

		for rows.Next() {
			var iCanLearn, iCanCrawl int
			var RERaw string

			err := rows.Scan(&RERaw, &canLearn, &canCrawl)
			if err != nil {
				log.Fatalf("Unable to process crawling restrictions/perms, %s", err.Error())
			}

			RE := regexp.MustCompile(RERaw)

			n := permissionRow{
				Pattern:  RE,
				CanCrawl: iCanCrawl == 0,
				CanLearn: iCanLearn == 0,
			}

			permissionsCache = append(permissionsCache, n)
		}
		log.Printf("%d perm restrictions loaded", len(permissionsCache))
	}

	// Now that it's all loaded...

	for _, testcase := range permissionsCache {
		if testcase.Pattern.MatchString(path) {
			return !testcase.CanCrawl, !testcase.CanLearn
		}
	}

	return true, true
}
