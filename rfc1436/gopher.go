package rfc1436

import (
	"fmt"
	"io/ioutil"
	"net"
	"time"
)

func Get(uri string) (out Document, info GopherConnectionInfo, err error) {
	bin, ci, err := GetBinary(uri)
	if err != nil {
		return out, GopherConnectionInfo{}, err
	}

	Doc, err := parseDocument(bin)
	if err != nil {
		return out, ci, err
	}

	return Doc, ci, err
}

func GetBinary(uri string) (out []byte, info GopherConnectionInfo, err error) {
	path, hostname, port, err := parseURI(uri)
	if err != nil {
		return out, GopherConnectionInfo{}, err
	}

	return requestRaw(path, hostname, port)
}

func requestRaw(path, hostname string, port int) (out []byte, info GopherConnectionInfo, err error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", hostname, port), time.Second*5)

	cinfo := GopherConnectionInfo{
		Path:     path,
		Hostname: hostname,
		Port:     port,
	}

	if err != nil {
		return out, cinfo, err
	}
	conn.SetReadDeadline(time.Now().Add(time.Second * 10))

	defer conn.Close()

	if path == "/" {
		path = ""
	}

	payload := fmt.Sprintf("%s\r\n", path)
	_, err = conn.Write([]byte(payload))
	if err != nil {
		return out, cinfo, err
	}

	out, err = ioutil.ReadAll(conn)
	return out, cinfo, err
}
