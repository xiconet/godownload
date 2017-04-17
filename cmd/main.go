// An HTTP parallel downloader client

package main

import (
	"github.com/alexflint/go-arg"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/xiconet/godownload"
	"os"
	"time"
	"strings"
	"net/url"
)

func DisplayProgress(dl *download.Downloader) {
	for {
		status, total, downloaded, elapsed := dl.GetProgress()
		fmt.Fprintf(os.Stdout, "Downloaded %.2f%% of %s, at %s/s\r", float64(downloaded)*100/float64(total), humanize.IBytes(uint64(total)), humanize.Bytes(uint64(float64(downloaded)/elapsed.Seconds())))
		switch {
		case status == download.Completed:
			fmt.Println("\nSuccessfully completed download in", elapsed)
			return
		case status == download.OnProgress:
		case status == download.NotStarted:
		default:
			fmt.Printf("\nFailed: %s\n", status)
			os.Exit(1)
		}
		time.Sleep(time.Second)
	}
}

func main() {
	var args struct {
		Url       string   `arg:"positional,help:url to fetch"`
		Conns     int      `arg:"-c,help:number of connections"`
		Outfile   string   `arg:"-o,help:optional <outfile> name"`
		Headers   []string `arg:"-H,help:custom headers as key=value pairs"`
	}

	args.Conns = 1
	arg.MustParse(&args)	
	if args.Url == "" {
		fmt.Println("usage error: missing <url> argument")
		os.Exit(1)
	}
	uri := u.String()	
	headers := map[string]string{}
	if len(args.Headers) == 0 {
		headers = nil 
	} else {
		for _, p := range args.Headers {
			pair := strings.Split(p, "=") 
			if len(pair) != 2 {
				fmt.Printf("usage error: invalid <headers> argument: %q\n", pair)
				os.Exit(1)
			}
			if pair[0] == "" || pair[1] == "" {
				fmt.Printf("usage error: invalid <headers> argument: %q\n", pair)
				os.Exit(1)
			}
			headers[pair[0]] = pair[1] 
		}
	}
	d := download.New()
	size, filename, err := d.Init(uri, args.Conns, args.Outfile)
	fmt.Printf("File size: %s; filename: %s\n", humanize.IBytes(size), filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	d.StartDownload()
	go d.Wait()
	DisplayProgress(&d)

}
