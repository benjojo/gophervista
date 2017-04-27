package rfc1436

import (
	"strconv"
	"strings"
)

// Gopher Message type codes
// See: https://www.ietf.org/rfc/rfc1436.txt
const (
	TypeTextFile         = "0"
	TypeMenuEntity       = "1"
	TypeCSOProtocol      = "2"
	TypeErr              = "3"
	TypeMacintoshBinHex  = "4"
	TypePCDOS            = "5"
	Typeuuencoded        = "6"
	TypeIndexServer      = "7"
	TypeTelnetSession    = "8"
	TypeBinaryFile       = "9"
	TypeDuplicatedServer = "+"
	TypeGIF              = "g"
	TypeImage            = "I"
	TypeTN3270           = "T"
	TypeHTML             = "h"
	TypeMessage          = "i"
	TypeWAVSound         = "s"
)

// Document repersents a Gopher document that has been
// parsed and ready to be processed for either
// reading or rendering
type Document struct {
	Raw   []byte
	Items []Item
}

type Item struct {
	Type    string
	Content string
	Host    string
	Port    int
	Path    string
}

func parseDocument(in []byte) (D Document, err error) {
	D.Raw = in
	D.Items = make([]Item, 0)

	lines := strings.Split(string(in), "\r\n")
	for _, line := range lines {
		if line == "." {
			break
		}

		if !strings.Contains(line, "\t") {
			continue
		}

		if len(line) < 5 {
			// too short to be a gopher listing
			continue
		}

		if !isValidGopherType(string(line[:1])) {
			// is unparsable
			continue
		}

		e := Item{}
		e.Type = string(line[:1])
		parts := strings.Split(line[1:], "\t")
		if len(parts) != 4 {
			// Unexpected amounts of tabs
			continue
		}

		e.Content = parts[0]
		e.Path = parts[1]
		e.Host = parts[2]
		pn, err := strconv.ParseInt(parts[3], 10, 16)
		if err != nil {
			// invalid port number
			continue
		}
		e.Port = int(pn)

		D.Items = append(D.Items, e)
	}
	return D, nil
}

func isValidGopherType(in string) bool {
	if in == "0" || in == "1" || in == "2" || in == "3" ||
		in == "4" || in == "5" || in == "6" || in == "7" ||
		in == "8" || in == "9" || in == "+" || in == "g" ||
		in == "I" || in == "T" || in == "h" || in == "i" ||
		in == "s" {
		return true
	}
	return false
}
