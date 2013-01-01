// An HTTP parallel downloader library

package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
	"errors"
)

const (
	NotStarted = "Not started"
	OnProgress = "On progress"
	Completed  = "Completed"
)

type Status string

type Downloader struct {
	url      string
	conns    int
	file     *os.File
	size     int
	parts    []Part
	start    time.Time
	end      time.Time
	done     chan error
	quit     chan bool
	status   string
}

func New() Downloader {
	return Downloader{}
}

func (dl *Downloader) Init(url string, conns int, filename string) (uint64, error) {
	dl.url = url
	dl.conns = conns
	dl.status = NotStarted
	resp, err := http.Head(url)
	if resp.StatusCode != 200 {
		return 0, errors.New(resp.Status)
	}
	if err != nil {
		return 0, err
	}
	dl.size, err = strconv.Atoi(resp.Header.Get("Content-Length"))

	if err != nil {
		return 0, errors.New("Not supported for download")
	}

	_, err = os.Stat(filename)
	if os.IsExist(err) {
		os.Remove(filename)
	}

	dl.file, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return 0, err
	}

	dl.parts = make([]Part, dl.conns)
	size := dl.size / dl.conns

	for i, part := range dl.parts {
		part.id = i
		part.url = dl.url
		part.offset = i * size
		switch {
		case i == dl.conns-1:
			part.size = dl.size - size*i
			break
		default:
			part.size = size
		}
		part.dlsize = 0
		part.file = dl.file
		dl.parts[i] = part
	}

	return uint64(dl.size), nil
}

func (dl *Downloader) StartDownload() {
	dl.done = make(chan error, dl.conns)
	dl.quit = make(chan bool, dl.conns)
	dl.status = OnProgress
	for i := 0; i < dl.conns; i++ {
		go dl.parts[i].Download(dl.done, dl.quit)
	}
	dl.start = time.Now()
}

func (dl Downloader) GetProgress() (status string, total, downloaded int, elapsed time.Duration) {
	dlsize := 0
	for _, part := range dl.parts {
		dlsize += part.dlsize
	}
	return dl.status, dl.size, dlsize, time.Now().Sub(dl.start)
}

func (dl *Downloader) Wait() error {
	var err error = nil
	for i:=0 ; i< dl.conns; i++ {
		e := <-dl.done
		if e != nil {
			err = e
			dl.status = err.Error()
			for j:=i ; j< dl.conns; j++ {
				dl.quit <- true
			}
		}
	}
	close(dl.done)
	dl.end = time.Now()
	dl.file.Close()
	if dl.status == OnProgress {
		dl.status = Completed
	}
	return err
}

func (dl *Downloader) Download() error {
	dl.StartDownload()
	return dl.Wait()
}

type Part struct {
	id     int
	url    string
	offset int
	dlsize int
	size   int
	file   *os.File
}

func (part *Part) Download(done chan error, quit chan bool) error {
	client := http.Client{}
	buffer := make([]byte, 4096)
	req, err := http.NewRequest("GET", part.url, nil)
	defer func() {
		done <- err
	}()

	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", part.offset, part.offset+part.size-1))
	resp, e := client.Do(req)
	err = e
	if err != nil {
		return err
	}

	for {
		select {
		case <- quit:
			return nil
		default:
		}
		/*
		if part.id == 5 {
			time.Sleep(time.Second)
			err = errors.New("Connection reset by peer")
			return err
		}
		*/
		nbytes, err := resp.Body.Read(buffer[0:])
		if err == io.EOF {
			resp.Body.Close()
			break
		}
		if err != nil {
			return err
		}

		nbytes, err = part.file.WriteAt(buffer[0:nbytes], int64(part.offset+part.dlsize))
		if err != nil {
			return nil
		}
		part.dlsize += nbytes

	}

	return nil
}