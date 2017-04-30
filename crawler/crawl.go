package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"encoding/base32"
	"io/ioutil"

	"github.com/benjojo/gophervista/rfc1436"
)

func crawl(Queue crawlerQueue, datadir string) {

	for {
		asset := Queue.GetItemToCrawl()
		if asset == "" {
			log.Print("Nothing to crawl...")
			continue
		}

		canCrawl, canLearn := Queue.CheckItemPerms(asset)

		if !canCrawl {
			log.Printf("Can't crawl %s because of restriction", asset)
			Queue.FlagItemAsCrawled(asset)
			continue
		}
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
			log.Printf("Unable to make directory structure (%s) to store responce in, %s")
		}

		ioutil.WriteFile(storefolder+base32shortcut(asset), d.Raw, 0755)

		for _, ln := range d.Items {
			if ln.Type == rfc1436.TypeMenuEntity ||
				ln.Type == rfc1436.TypeTextFile {
				newpath := fmt.Sprintf("gopher://%s:%d%s", ln.Host, ln.Port, ln.Path)
				if canLearn {
					Queue.FlagItem(newpath, ln.Type)
					log.Printf("Found item to add %s", newpath)
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
