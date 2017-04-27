package rfc1436

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func parseURI(inputURI string) (path, hostname string, port int, err error) {
	if strings.HasPrefix(inputURI, "gopher://") {
		U, err := url.Parse(inputURI)
		if err != nil {
			return "", "", 0, err
		}

		// Now figure the port out
		if strings.Contains(U.Host, ":") {
			parts := strings.Split(U.Host, ":")
			if len(parts) != 2 {
				return "", "", 0, fmt.Errorf("the URI provided has more than one colon")
			}

			p, err := strconv.ParseUint(parts[1], 10, 16)
			if err != nil {
				return "", "", 0, fmt.Errorf("port on URI is invalid")
			}
			return U.Path, U.Host, int(p), nil
		}
		return U.Path, U.Host, 70, nil
	}
	return parseURI("gopher://" + inputURI)
}
