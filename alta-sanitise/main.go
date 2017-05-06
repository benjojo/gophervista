package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/benjojo/gophervista/rfc1436"
	_ "github.com/mattn/go-sqlite3"
)

var (
	listen  = flag.String("listen", ":9982", "Where to listen on")
	backend = flag.String("backend", "localhost:5555", "Where to listen on")
	dbpath  = flag.String("dbpath", "../crawl.db", "Where to find the database")
)

func main() {
	flag.Parse()

	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/query", backendSearchEntry)
	http.ListenAndServe(*listen, http.DefaultServeMux)
}

var titlematcher = regexp.MustCompile("<strong>(\\d+)\\.txt</strong>")
var urlmatcher = regexp.MustCompile("href=(http[^>]+)")
var url2matcher = regexp.MustCompile(`(d:\\\d+\\\d+.txt)`)
var url3matcher = regexp.MustCompile("<dd><a href=(http[^>]+)")

func backendSearchEntry(rw http.ResponseWriter, req *http.Request) {
	if req.URL.Query().Get("q") == "" {
		http.Redirect(rw, req, "/", http.StatusTemporaryRedirect)
		return
	}

	stq := 0

	if req.URL.Query().Get("stq") != "" {
		i, err := strconv.ParseInt(req.URL.Query().Get("stq"), 10, 64)
		if err == nil {
			stq = int(i)
		}
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(fmt.Sprintf("https://%s/?pg=q&what=0&q=%s&stq=%d", *backend, url.QueryEscape(req.URL.Query().Get("q")), stq))
	if err != nil {
		http.Error(rw, "failed to get back out of the backend, may god help you", http.StatusInternalServerError)
		return
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(rw, "backend stopped serving halfway though, god help you and me", http.StatusInternalServerError)
		return
	}

	db, err := sql.Open("sqlite3", *dbpath)
	if err != nil {
		http.Error(rw, "db failed, god help you", http.StatusInternalServerError)
		return
	}

	err = db.Ping() // database do odd things
	if err != nil {
		http.Error(rw, "db is broken, god help me", http.StatusInternalServerError)
		return
	}

	htmlpage := string(b)
	// Fix up some of the URL and titles
	htmlpage = strings.Replace(htmlpage, "file:///C:\\Program Files\\DIGITAL\\AltaVista Search\\My Computer\\images\\", "/images/", -1)
	htmlpage = strings.Replace(htmlpage, "action=\"http://127.0.0.1:6688\"", "action=\"/query\"", -1)
	htmlpage = strings.Replace(htmlpage, "<title>AltaVista Personal 97</title>", "<title>GopherVista 97 - The gopher search engine</title>", -1)
	htmlpage = strings.Replace(htmlpage, "<OPTION value=0 SELECTED>My Computer All", "<OPTION value=0 SELECTED>Gopherspace", -1)
	htmlpage = strings.Replace(htmlpage, "http://127.0.0.1:6688/?pg=q&fmt=d", "/query?", -1)
	htmlpage = strings.Replace(htmlpage, "http://127.0.0.1:6688/?pg=aq", "/?pq", -1)
	htmlpage = strings.Replace(htmlpage, "http://127.0.0.1:6688/?pg=q\"", "/?pq\"", -1)
	htmlpage = strings.Replace(htmlpage, "http://127.0.0.1:6688/?pg=h", "/?pq", -1)
	htmlpage = strings.Replace(htmlpage, "http://127.0.0.1:6688/?pg=config&what=init", "/?pq", -1)
	htmlpage = strings.Replace(htmlpage, `href="http://www.digital.com:80/"`, "href=/", -1)

	// Now let's remove some of the crap
	removestrings := []string{
		"<OPTION value=-1 >Netscape Cache",
		"<OPTION value=2 >My Computer Documents",
		"<OPTION value=1 >My Computer Mail",
		"<OPTION value=4 > Usenet  [US]",
		"<OPTION value=3 > the Web  [US]",
		"<p>Click <A href=\"http://support.altavista.software.digital.com/ISBUTECHSUP/intro.htm\"> here</a> to contact AltaVista Support.</a>",
		"<img src=\"/images/static_banner.gif\" width=468 height=60 border=0 alt=\"[Digital Equipment Corporation]\">",
	}

	for _, killstr := range removestrings {
		htmlpage = strings.Replace(htmlpage, killstr, "", -1)
	}

	// Now to start rewriting things with things that are in the DB
	lines := strings.Split(htmlpage, "\n")
	for _, line := range lines {
		if strings.Contains(line, "<dt><font size=-1>") {
			title := titlematcher.FindAllStringSubmatch(line, 1)
			if len(title) != 1 {
				htmlpage = strings.Replace(htmlpage, line, "", -1)
				continue
			}

			dbid, _ := strconv.ParseInt(title[0][1], 10, 64)
			gopherstr := ""
			datatype := 0
			err := db.QueryRow("SELECT path,type FROM assets WHERE id=? LIMIT 1", int(dbid)).Scan(&gopherstr, &datatype)
			if err != nil {
				htmlpage = strings.Replace(htmlpage, line, "", -1)
				continue
			}
			p, hn, port, _ := rfc1436.ParseURI(gopherstr)
			gopherstr = fmt.Sprintf("gopher://%s:%d/%d%s", hn, port, datatype, p)

			htmlpage = strings.Replace(htmlpage, fmt.Sprintf("<strong>%d.txt", dbid), fmt.Sprintf("<strong>%s", gopherstr), -1)

			url1 := url2matcher.FindAllStringSubmatch(line, 1)
			if len(url1) != 1 {
				htmlpage = strings.Replace(htmlpage, line, "", -1)
				continue
			}
			htmlpage = strings.Replace(htmlpage, url1[0][1], gopherstr, -1)

			urls := urlmatcher.FindAllStringSubmatch(line, 1)
			htmlpage = strings.Replace(htmlpage, urls[0][1], gopherstr, 1)
			urls = url3matcher.FindAllStringSubmatch(line, 2)
			htmlpage = strings.Replace(htmlpage, urls[0][1], fmt.Sprintf("http://gopher.floodgap.com/gopher/gw?%s", url.QueryEscape(gopherstr)), 1)
		}
	}
	rw.Write([]byte(htmlpage))
}
