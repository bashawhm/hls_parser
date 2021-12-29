package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type MediaFile struct {
	bandwidth   uint64
	codecs      string
	resolutionX uint64
	resolutionY uint64
	name        string
	url         string
}

type HLSPlaylist struct {
	files []MediaFile
}

func splitDirective(dir string, delimeter rune) (parts []string) {
	inSubString := false
	lastSegment := 0

	for i := 0; i < len(dir); i++ {
		if inSubString {
			if dir[i] == []byte("\"")[0] {
				inSubString = false
			}
			continue
		}

		if dir[i] == []byte("\"")[0] {
			inSubString = true
		}

		if dir[i] == byte(delimeter) {
			segment := dir[lastSegment:i]
			parts = append(parts, segment)
			lastSegment = i + 1
			continue
		}
	}

	segment := dir[lastSegment:]
	parts = append(parts, segment)

	return
}

func HLSPlaylistParse(text string) (out HLSPlaylist) {
	streamInfMatch, _ := regexp.Compile("EXT-X-STREAM-INF:*")
	bandwidthMatch, _ := regexp.Compile("BANDWIDTH=([0-9]+)")
	codecMatch, _ := regexp.Compile("CODECS=")
	nameMatch, _ := regexp.Compile("NAME=")
	resMatch, _ := regexp.Compile("RESOLUTION=")

	directives := strings.Split(text, "#")

	for i := 0; i < len(directives); i++ {
		if streamInfMatch.MatchString(directives[i]) {
			var media MediaFile

			parts := strings.Split(directives[i], "\n")

			chunks := splitDirective(parts[0], ',')
			for j := 0; j < len(chunks); j++ {
				if bandwidthMatch.MatchString(chunks[j]) {
					sections := strings.Split(chunks[j], "=")
					media.bandwidth, _ = strconv.ParseUint(sections[len(sections)-1], 10, 64)
				}
				if nameMatch.MatchString(chunks[j]) {
					sections := strings.Split(chunks[j], "=")
					media.name = sections[len(sections)-1]
				}
				if codecMatch.MatchString(chunks[j]) {
					sections := strings.Split(chunks[j], "=")
					media.codecs = sections[len(sections)-1]
				}
				if resMatch.MatchString(chunks[j]) {
					sections := strings.Split(chunks[j], "=")
					reses := strings.Split(sections[len(sections)-1], "x")
					media.resolutionX, _ = strconv.ParseUint(reses[0], 10, 64)
					media.resolutionY, _ = strconv.ParseUint(reses[1], 10, 64)
				}
			}

			media.url = parts[1]

			out.files = append(out.files, media)
		}
	}

	return
}

func main() {
	resp, err := http.Get("https://test-streams.mux.dev/x36xhzz/x36xhzz.m3u8")
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil && resp.StatusCode > 299 {
		panic(err)
	}

	fmt.Println(string(body))

	playlist := HLSPlaylistParse(string(body))
	fmt.Println("{")
	for i := 0; i < len(playlist.files); i++ {
		fmt.Println("\t", playlist.files[i])
	}
	fmt.Println("}")
}
