package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Download struct {
	Url           string
	TargetPath    string
	TotalSections int
	Size          int64
}

func (d *Download) Do(progressChan chan int64) error {
	r, err := d.makeRequest("HEAD")
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}

	if resp.StatusCode > 299 {
		return errors.New(fmt.Sprintf("Can't process, response is %d", resp.StatusCode))
	}

	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return err
	}

	d.Size = size
	fmt.Printf("Size: %s\n", formatBytes(size))

	sections := make([][2]int64, d.TotalSections)
	eachSize := d.Size / int64(d.TotalSections)

	for i := range sections {
		if i == 0 {
			sections[i][0] = 0
		} else {
			sections[i][0] = sections[i-1][1] + 1
		}

		if i < d.TotalSections-1 {
			sections[i][1] = sections[i][0] + eachSize
		} else {
			sections[i][1] = d.Size - 1
		}
	}

	err = os.Mkdir("temp", os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	var wg sync.WaitGroup

	for i, s := range sections {
		wg.Add(1)
		i, s := i, s
		go func() {
			defer wg.Done()
			err = d.downloadSection(i, s, progressChan)
			if err != nil {
				fmt.Printf("Error downloading section %d: %v\n", i, err)
			}
		}()
	}

	wg.Wait()
	close(progressChan)

	err = d.mergeFile(sections)
	if err != nil {
		return err
	}

	err = os.RemoveAll("temp")
	if err != nil {
		return err
	}

	return nil
}

func (d *Download) makeRequest(method string) (*http.Request, error) {
	r, err := http.NewRequest(method, d.Url, nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "Downloader")

	return r, nil
}

func (d *Download) downloadSection(i int, sections [2]int64, progressChan chan int64) error {
	sectionPath := fmt.Sprintf("temp/section-%d.tmp", i)
	file, err := os.OpenFile(sectionPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	downloaded := fi.Size()
	progressChan <- downloaded
	sections[0] += downloaded

	r, err := d.makeRequest("GET")
	if err != nil {
		return err
	}

	r.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", sections[0], sections[1]))
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}

	if resp.StatusCode > 299 {
		return errors.New(fmt.Sprintf("Status code: %d", resp.StatusCode))
	}

	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, err := file.Write(buf[:n])
			if err != nil {
				return err
			}
			progressChan <- int64(n)
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Download) mergeFile(sections [][2]int64) error {
	f, err := os.OpenFile(d.TargetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	for i := range sections {
		b, err := ioutil.ReadFile(fmt.Sprintf("temp/section-%d.tmp", i))
		if err != nil {
			return err
		}
		_, err = f.Write(b)
		if err != nil {
			return err
		}
	}

	return nil
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func main() {
	url := flag.String("url", "", "URL to download")
	flag.Parse()

	if *url == "" {
		fmt.Println("Please specify the URL to download")
		return
	}

	fileName := strings.Split(*url, "/")[len(strings.Split(*url, "/"))-1]
	startTime := time.Now()

	numCores := runtime.NumCPU()
	runtime.GOMAXPROCS(numCores)

	d := &Download{
		Url:           *url,
		TargetPath:    fileName,
		TotalSections: numCores,
	}

	progressChan := make(chan int64)
	go func() {
		totalDownloaded := int64(0)
		for bytesRead := range progressChan {
			totalDownloaded += bytesRead

			elapsed := time.Since(startTime)
			downloadSpeed := float64(totalDownloaded) / elapsed.Seconds()
			percent := float64(totalDownloaded) / float64(d.Size) * 100
			remainingTime := time.Duration((float64(d.Size-totalDownloaded) / downloadSpeed))

			fmt.Printf("\rDownloaded: %s/%s (%.2f%%) | Speed: %s/s | Elapsed: %s | Remaining: %s",
				formatBytes(totalDownloaded), formatBytes(d.Size), percent, formatBytes(int64(downloadSpeed)),
				formatDuration(elapsed), formatDuration(remainingTime))
		}
	}()

	err := d.Do(progressChan)
	if err != nil {
		fmt.Println("\nError:", err)
		return
	}

	fmt.Printf("\nTime taken: %s\n", formatDuration(time.Since(startTime)))
}
