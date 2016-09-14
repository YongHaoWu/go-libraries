package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
)

// Possible states.
const (
	Stopped = iota
	Running
	Finished
)

type PassThru struct {
	io.Reader
	skillName      string
	totalSize      int64
	downloadedSize int64
	status         int
	statusWS       chan int
}

func main() {
	urls := []string{"http://av.jejeso.com/robotcenter", "http://av.jejeso.com/small.pdf"}
	names := []string{"robotcenter", "small.pdf"}

	downloadWorkders := make([]chan int, len(urls))
	passThruGroup := make([]PassThru, len(urls))

	for i := range downloadWorkders {
		downloadWorkders[i] = make(chan int, 1)
		downloadWorkders[i] <- Running

		passThruGroup[i] = PassThru{statusWS: downloadWorkders[i], skillName: names[i]}
		go downloadFile(names[i], urls[i], &passThruGroup[i])

	}
	time.Sleep(5000 * time.Millisecond)
	downloadWorkders[0] <- Stopped

	var input string
	fmt.Scanln(&input)
	fmt.Println("done")
}

// Read 'overrides' the underlying io.Reader's Read method.
// This is the one that will be called by io.Copy(). We simply
// use it to keep track of byte counts and then forward the call.
func (pt *PassThru) Read(p []byte) (int, error) {
	for {
		select {
		case pt.status = <-pt.statusWS:
			switch pt.status {
			case Stopped:
				fmt.Printf(pt.skillName + "Stopped\n")
			case Running:
				fmt.Printf(pt.skillName + " Running\n")
			}
		default:
			runtime.Gosched()
			if pt.status == Stopped {
				break
			}
			n, err := pt.Reader.Read(p)
			pt.downloadedSize += int64(n)
			if err == nil {
				fmt.Println(pt.skillName+"status is ", pt.status, " Read", n, "bytes for a downloadedSize of", pt.downloadedSize,
					"pt.totalSize", pt.totalSize)
				fmt.Printf("percent is %.4f\n", float64(pt.downloadedSize)/float64(pt.totalSize))
			}
			return n, err
		}
	}
	return 0, nil
}

func downloadFile(filepath string, url string, passThru *PassThru) (err error) {
	// TODO: check file existence first with io.IsExist
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var src io.Reader
	passThru.Reader = resp.Body
	src = passThru
	fmt.Println("start")
	passThru.totalSize = resp.ContentLength
	if _, err = io.Copy(out, src); err != nil {
		return err
	}
	fmt.Println("end close")
	return nil
}
