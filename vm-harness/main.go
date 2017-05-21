package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	qemucmd := exec.Command("/usr/local/bin/qemu-system-x86_64",
		"-m", "192",
		"-drive", "file=98.qcow2,id=disk,cache=unsafe",
		"-net", "nic,model=ne2k_isa",
		"-net", "user,hostfwd=tcp:127.0.0.1:5555-:8443",

		"-drive", "file=0.iso,index=3,media=cdrom",
		"-drive", "file=1.iso,index=1,media=cdrom",
		"-drive", "file=2.iso,index=2,media=cdrom",

		"-vnc", "127.0.0.1:0",
		"-vga", "cirrus",
		"-serial", "stdio") // to read off the windows 98 reporter

	qemucmd.Stderr = os.Stderr
	serial, err := qemucmd.StdoutPipe()
	if err != nil {
		log.Fatalf("!!! %s", err.Error())
	}

	c, err := net.Dial("unix", "/var/run/collectd.socket")
	if err != nil {
		log.Fatalf("Cannot connect to collectd! %s", err.Error())
	}

	go func() {
		err := qemucmd.Run()
		if err != nil {
			log.Fatalf("!!! %s", err.Error())
		}
	}()

	time.Sleep(time.Second * 5)

	go func() {
		for {
			b := make([]byte, 1000)
			n, err := c.Read(b)
			if err != nil {
				return
			}
			fmt.Print(string(b[:n]))
		}
	}()

	reader := bufio.NewReader(serial)
	for {
		ln, _, err := reader.ReadLine()
		if err != nil {
			return
		}
		c.Write([]byte(strings.Replace(string(ln), " N: ", " N:", 1) + "\n"))
	}
}
