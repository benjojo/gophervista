package main

import (
	"encoding/base32"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/benjojo/gophervista/rfc1436"
)

func crawl(Queue crawlerQueue, datadir string) {
	var workers map[string]chan string
	workers = make(map[string]chan string)
	indexn := 0
	skiphost := "impossibleaaaaaaaaaaaaaaaaaa"

	for {
		time.Sleep(time.Millisecond * 5)
		var asset string
		asset, indexn = Queue.GetItemToCrawlFromIndex(indexn, skiphost)
		if asset == "" {
			log.Print("Nothing to crawl...")
			indexn = 0
			continue
		}
		indexn++

		canCrawl, _ := Queue.CheckItemPerms(asset)

		if !canCrawl {
			log.Printf("Can't crawl %s because of restriction", asset)
			Queue.FlagItemAsCrawled(asset)
			continue
		}

		_, hn, _, err := rfc1436.ParseURI(asset)
		if err != nil {
			log.Printf("Invalid URI, %s", err.Error())
			continue
		}

		if workers[hn] == nil {
			ch := make(chan string, 1)
			workers[hn] = ch
			go crawlWorker(ch, Queue, datadir)
		}
		select {
		case workers[hn] <- asset: // ignore if it's full.
			skiphost = "impossibleaaaaaaaaaaa"
		default:
			skiphost = hn
		}

	}
}

func crawlWorker(assetChan chan string, Queue crawlerQueue, datadir string) {
	for asset := range assetChan {
		if asset == "" {
			log.Print("Nothing to crawl...")
			continue
		}

		_, canLearn := Queue.CheckItemPerms(asset)

		time.Sleep(time.Second)

		d, ci, err := rfc1436.Get(asset)
		log.Printf("Grabbing %s", asset)
		if err != nil {
			log.Printf("Failed to crawl '%s' - %s", asset, err.Error())
			Queue.FlagItemAsCrawled(asset)
			continue
		}

		storefolder := fmt.Sprintf("%s/raw/%s-%d/", datadir, ci.Hostname, ci.Port)
		err = os.MkdirAll(storefolder, 0755)
		if err != nil && err != os.ErrExist {
			log.Printf("Unable to make directory structure (%s) to store responce in, %s", storefolder, err.Error())
		}

		ioutil.WriteFile(storefolder+base32shortcut(asset), d.Raw, 0755)

		for _, ln := range d.Items {
			if ln.Type == rfc1436.TypeMenuEntity ||
				ln.Type == rfc1436.TypeTextFile {
				newpath := fmt.Sprintf("gopher://%s:%d%s", ln.Host, ln.Port, ln.Path)
				if canLearn {
					Queue.FlagItem(newpath, ln.Type)
				} else {
					log.Printf("Can't add %s to crawl list, Domain has restriction in place", newpath)
				}
			}
		}
		Queue.FlagItemAsCrawled(asset)
	}
}

func base32shortcut(path string) string {
	return base32.StdEncoding.EncodeToString([]byte(path))
}
