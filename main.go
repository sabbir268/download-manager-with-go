package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// Language: go
// Path: main.go

type Download struct {
	Url           string
	TargetPath    string
	TotalSections int
}

func (d Download) Do() error {
	fmt.Println("Downloading")
	r, err := d.makeRequest("HEAD")
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}

	if resp.StatusCode > 299 {
		return errors.New(fmt.Sprint("Status code:", resp.StatusCode))
	}
	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))

	if err != nil {
		return err
	}

	fmt.Println("Size:", size)

	sections := make([][2]int, d.TotalSections)
	eachSize := size / d.TotalSections

	// for i := 0; i < d.TotalSections; i++ {
	// 	sections[i] = [2]int{i * eachSize, (i + 1) * eachSize}
	// }
	for i := range sections {
		if i == 0 {
			// starting byte of first section
			sections[i][0] = 0
		} else {
			// starting byte of other sections
			sections[i][0] = sections[i-1][1] + 1
		}

		if i < d.TotalSections-1 {
			// ending byte of other sections
			sections[i][1] = sections[i][0] + eachSize
		} else {
			// ending byte of other sections
			sections[i][1] = size - 1
		}
	}

	var wg sync.WaitGroup

	for i, s := range sections {
		fmt.Println("Section", i, ":", s)
		wg.Add(1)
		i := i
		s := s
		go func() {
			defer wg.Done()
			err = d.downloadSection(i, s)

			if err != nil {
				panic(err)
			}
		}()
	}
	wg.Wait()
	err = d.mergeFile(sections)
	if err != nil {
		return err
	}
	return nil
}

func (d Download) makeRequest(method string) (*http.Request, error) {
	r, err := http.NewRequest(method, d.Url, nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DMMS")

	return r, nil
}

func (d Download) downloadSection(i int, sections [2]int) error {
	r, err := d.makeRequest("GET")
	if err != nil {
		return err
	}

	r.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", sections[0], sections[1]))
	// fmt.Println("r", r)
	resp, err := http.DefaultClient.Do(r)

	if err != nil {
		return err
	}

	if resp.StatusCode > 299 {
		return errors.New(fmt.Sprint("Status code:", resp.StatusCode))
	}
	fmt.Println("Content Length:", resp.Header.Get("Content-Length"))
	fmt.Println("Downloading section", i)

	br, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fmt.Sprintf("section-%v.tmp", i), br, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func (d Download) mergeFile(sections [][2]int) error {
	f, err := os.OpenFile(d.TargetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	for i := range sections {
		fmt.Println("Merging section", i)
		b, err := ioutil.ReadFile(fmt.Sprintf("section-%v.tmp", i))
		if err != nil {
			return err
		}
		n, err := f.Write(b)
		if err != nil {
			return err
		}

		fmt.Println("Wrote", n, "bytes")
	}

	return nil
}

func main() {
	startTime := time.Now()

	d := Download{
		Url: "https://sample-videos.com/video123/mp4/720/big_buck_bunny_720p_5mb.mp4",
		// Url:           "http://172.16.50.8/SAM-FTP/English%20Movies%20%281080p%29/%282021%29%201080p/Shang-Chi%20and%20the%20Legend%20of%20the%20Ten%20Rings%20%282021%29%201080p/Shang-Chi%20and%20the%20Legend%20of%20the%20Ten%20Rings%20%282021%29%201080p%20BluRay%20x265%20HEVC%2010bit%20AAC%207.1%20MSubs-PSA.mkv",
		TargetPath:    "test.mp4",
		TotalSections: 10,
	}

	err := d.Do()

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Time taken:", time.Since(startTime))

}
