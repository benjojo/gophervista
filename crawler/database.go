package main

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"
)

type crawlerQueue struct {
	db       *sql.DB
	dblock   sync.Mutex
	flagchan chan FlagRequest
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

	cq.flagchan = make(chan FlagRequest, 1)
	go cq.FlagChannelConsumer(cq.flagchan)

	return nil
}

func (cq *crawlerQueue) maketableifnotexists(table, makestring string) {
	tablename := ""
	cq.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", table).Scan(&tablename)

	if tablename != table {
		log.Printf("Made a new %s table", table)
		cq.db.Exec(makestring)
	}

}

func (cq *crawlerQueue) GetItemToCrawl() string {
	cq.dblock.Lock()
	defer cq.dblock.Unlock()
	asset := ""
	err := cq.db.QueryRow("SELECT path FROM assets WHERE lastcrawled < ? LIMIT 1", time.Now().Unix()-604800).Scan(&asset)
	if err != nil {
		return ""
	}
	return asset
}

func (cq *crawlerQueue) GetItemToCrawlFromIndex(indexin int, skiphost string) (asset string, index int) {
	cq.dblock.Lock()
	defer cq.dblock.Unlock()
	err := cq.db.QueryRow("SELECT path,id FROM assets WHERE lastcrawled < ? AND id > ? AND path NOT LIKE ? LIMIT 1",
		time.Now().Unix()-604800, indexin, fmt.Sprintf("%%%s%%", skiphost)).Scan(&asset, &index)
	if err != nil {
		return "", 0
	}
	return asset, index
}

func (cq *crawlerQueue) GetItemToCrawlFromIndexFromHost(indexin int, targethost string) (asset string, index int) {
	cq.dblock.Lock()
	defer cq.dblock.Unlock()
	err := cq.db.QueryRow("SELECT path,id FROM assets WHERE lastcrawled < ? AND id > ? AND path LIKE ? LIMIT 1",
		time.Now().Unix()-604800, indexin, fmt.Sprintf("%%%s%%", targethost)).Scan(&asset, &index)
	if err != nil {
		return "", 0
	}
	return asset, index
}

func (cq *crawlerQueue) FlagItemAsCrawled(path string) {
	_, err := cq.db.Exec("UPDATE assets SET lastcrawled = ? WHERE path = ?", time.Now().Unix(), path)
	if err != nil {
		log.Printf("Unable to flag asset as crawled? %s", err.Error())
		if err.Error() == "database is locked" { // aaaa a aaaa aaaaaaaa terrible i'm so sorry
			log.Printf("Retrying in a moment due to %s", err.Error())
			go func() {
				// Well we are already doing sins here, so what's one more?
				time.Sleep(time.Second)
				cq.FlagItemAsCrawled(path)
			}()
		}
	}
}

type FlagRequest struct {
	Path     string
	GTypeStr string
}

func (cq *crawlerQueue) FlagItem(path string, gtypestr string) {
	cq.flagchan <- FlagRequest{
		Path:     path,
		GTypeStr: gtypestr,
	}
}

func (cq *crawlerQueue) FlagChannelConsumer(in chan FlagRequest) {
	for rq := range in {
		cq.dblock.Lock()
		test := 0
		err := cq.db.QueryRow("SELECT COUNT(*) path FROM assets WHERE path = ?;", rq.Path).Scan(&test)
		if err != nil {
			log.Printf("Huh, unable to see if an item to be should be crawled -  %s", err.Error())
			continue
		}

		if test == 0 {
			log.Printf("Found item to add %s", rq.Path)

			// Fantastic, We can add it
			_, err = cq.db.Exec("INSERT INTO `assets` (`id`,`timestamp`,`path`,`type`,`lastcrawled`) VALUES (NULL,?,?,?,0);", time.Now().Unix(), rq.Path, rq.GTypeStr)
			if err != nil {
				log.Printf("Huh, unable to flag to be should be crawled -  %s", err.Error())
				continue
			}
		}
		cq.dblock.Unlock()
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
