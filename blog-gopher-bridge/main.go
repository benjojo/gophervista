package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"

	"io"

	"github.com/benjojo/gophervista/rfc1436"
)

var (
	blogbasepath       = flag.String("blogpath", "https://blog.benjojo.co.uk", "Put the blog URL in here")
	gopherhostname     = flag.String("gopherhost", "gopher.blog.benjojo.co.uk", "The target host of the links generated")
	port               = flag.Int("port", 70, "port to listen on")
	errBlogFail        = fmt.Errorf("Failure in blog fetch")
	markdownImgRegexp  = regexp.MustCompile(`!\[([^\]]+)\]\(([^)]+)\)`)
	markdownLinkRegexp = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	listingLinkRegexp  = regexp.MustCompile(`<a href="/post/([^"]+)">(.+)`)
)

func main() {
	log.Printf("a")
	flag.Parse()
	l, err := net.Listen("tcp", ":70")
	if err != nil {
		log.Fatalf("unable to listen on gopher %s", err.Error())
	}

	for {
		c, err := l.Accept()
		if err != nil {
			continue
		}

		go handleGopherConnection(c)
	}
}

func handleGopherConnection(c net.Conn) {
	defer c.Close()

	bio := bufio.NewReader(c)
	request, _, err := bio.ReadLine()
	if err != nil {
		return
	}

	log.Printf("Accepted request from %s for %s", c.RemoteAddr().String(), string(request))

	if strings.HasPrefix(string(request), "/asset/") {
		getAssetContent(string(request), c)
		return
	}

	if string(request) == "" {
		getPostListings(c)
		return
	}

	blogpost, err := getBlogPost(string(request))
	if err != nil {
		c.Write([]byte(rfc1436.TypeErr + `Failed to get that blog post   fake    fake    0
`))
		log.Printf("Could not get blog post due to %s", err.Error())
		return
	}
	outputlines := make([]string, 0)

	lines := strings.Split(string(blogpost), "\n")
	for _, line := range lines {
		if len(markdownImgRegexp.FindAllStringSubmatch(line, -1)) != 0 {
			images := markdownImgRegexp.FindAllStringSubmatch(line, -1)
			for _, i := range images {
				line = strings.Replace(line, i[0], i[1], 1)
				path := strings.Replace(i[2], "https://blog.benjojo.co.uk/", "/", 1)
				outputlines = append(outputlines, fmt.Sprintf(rfc1436.TypeImage+"%s\t%s\t%s\t%d", i[1], path, *gopherhostname, *port))
			}
		}
		if len(markdownLinkRegexp.FindAllStringSubmatch(line, -1)) != 0 {
			links := markdownLinkRegexp.FindAllStringSubmatch(line, -1)
			for _, i := range links {
				line = strings.Replace(line, i[0], i[1], 1)
				url := "URL:" + strings.Replace(i[2], "https://", "http://", 1)
				outputlines = append(outputlines, fmt.Sprintf("h%s\t%s\t%s\t0", i[1], url, url))
			}
		}

		words := strings.Split(line, " ")
		linechars := 0
		linebuffer := ""
		for _, word := range words {
			if linechars+len(word)+2 > 70 {
				linechars = 0
				outputlines = append(outputlines, fmt.Sprintf(rfc1436.TypeMessage+"%s\tfake\tfake\t0", linebuffer))
				linebuffer = word
			} else {
				linechars += len(word) + 2
				linebuffer = linebuffer + " " + word
			}
		}
		outputlines = append(outputlines, fmt.Sprintf(rfc1436.TypeMessage+"%s\tfake\tfake\t0", linebuffer))
	}

	for _, ln := range outputlines {
		c.Write([]byte(ln + "\r\n"))
	}
}

func getBlogPost(path string) (blog string, err error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/raw%s", *blogbasepath, path), nil)
	req.Header.Set("User-Agent", "gopher-blog-bridge")
	cl := http.Client{}
	res, err := cl.Do(req)
	if err != nil {
		return "", err
	}

	blogbytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		return "", errBlogFail
	}

	return string(blogbytes), nil
}

func getAssetContent(path string, c net.Conn) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s%s", *blogbasepath, path), nil)
	req.Header.Set("User-Agent", "gopher-blog-bridge")
	cl := http.Client{}
	res, _ := cl.Do(req)
	io.Copy(c, res.Body)
}

func getPostListings(c net.Conn) {
	req, _ := http.NewRequest("GET", *blogbasepath, nil)
	req.Header.Set("User-Agent", "gopher-blog-bridge")
	cl := http.Client{}
	res, err := cl.Do(req)
	if err != nil {
		return
	}

	blogbytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	c.Write([]byte(fmt.Sprintf(rfc1436.TypeMessage + "Welcome to Gopher blog.Benjojo.co.uk\tfake\tfake\t0\r\n")))

	lines := strings.Split(string(blogbytes), "\n")
	for _, ln := range lines {
		if len(listingLinkRegexp.FindAllStringSubmatch(ln, -1)) != 0 &&
			!strings.Contains(ln, "</a>") {
			lnk := listingLinkRegexp.FindAllStringSubmatch(ln, -1)
			for _, i := range lnk {
				translate := strings.Replace(i[1], "/post", "", 1)
				c.Write([]byte(fmt.Sprintf(rfc1436.TypeMenuEntity+"%s\t%s\t%s\t%d\r\n", i[2], "/"+translate, *gopherhostname, *port)))
			}
		}

	}
}
