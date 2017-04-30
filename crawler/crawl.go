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
		time.Sleep(time.Second)
		asset := Queue.GetItemToCrawl()
		if asset == "" {
			log.Print("Nothing to crawl...")
			continue
		}

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
				log.Printf("Found item to add %s", newpath)
				Queue.FlagItem(newpath, ln.Type)
			}
		}
		Queue.FlagItemAsCrawled(asset)

	}

}

func base32shortcut(path string) string {
	return base32.StdEncoding.EncodeToString([]byte(path))
}
