package rfc1436

import (
	"fmt"
	"io/ioutil"
	"net"
)

func Get(uri string) (out Document, err error) {
	bin, err := GetBinary(uri)
	if err != nil {
		return out, err
	}

	Doc, err := parseDocument(bin)
	if err != nil {
		return out, err
	}

	return Doc, err
}

func GetBinary(uri string) (out []byte, err error) {
	path, hostname, port, err := parseURI(uri)
	if err != nil {
		return out, err
	}

	return requestRaw(path, hostname, port)
}

func requestRaw(path, hostname string, port int) (out []byte, err error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", hostname, port))
	defer conn.Close()
	if err != nil {
		return out, err
	}

	payload := fmt.Sprintf("%s\r\n", path)
	_, err = conn.Write([]byte(payload))
	if err != nil {
		return out, err
	}

	out, err = ioutil.ReadAll(conn)
	return out, err
}
