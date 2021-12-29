package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type MediaFile struct {
	durration float64
	mediaUrl  string
	data      []byte
}

type Playlist struct {
	bandwidth   uint64
	codecs      string
	resolutionX uint64
	resolutionY uint64
	name        string
	baseUrl     string
	playlistUrl string
	streamType  string
	segments    []MediaFile
}

type HLSMasterPlaylist struct {
	varients []Playlist
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

func (m *MediaFile) ResolveData() {
	resp, err := http.Get(m.mediaUrl)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil && resp.StatusCode > 299 {
		panic(err)
	}
	m.data = body
}

func (pl *Playlist) HLSPlaylistParse() {
	extInfMatch, _ := regexp.Compile("EXTINF:*")
	streamTypeMatch, _ := regexp.Compile("EXT-X-PLAYLIST-TYPE:*")

	resp, err := http.Get(pl.playlistUrl)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil && resp.StatusCode > 299 {
		panic(err)
	}

	text := string(body)
	directives := strings.Split(text, "#")
	for i := 0; i < len(directives); i++ {
		if streamTypeMatch.MatchString(directives[i]) {
			chunks := strings.Split(directives[i], ":")
			pl.streamType = strings.TrimSuffix(chunks[len(chunks)-1], "\n")
		}
		if extInfMatch.MatchString(directives[i]) {
			var media MediaFile

			parts := strings.Split(directives[i], "\n")

			chunks := splitDirective(parts[0], ',')
			durr := strings.TrimPrefix(chunks[0], "EXTINF:")
			media.durration, _ = strconv.ParseFloat(durr, 64)

			media.mediaUrl = pl.baseUrl + parts[1]

			pl.segments = append(pl.segments, media)
		}
	}

}

func HLSMasterPlaylistParse(text string, url string) (out HLSMasterPlaylist) {
	streamInfMatch, _ := regexp.Compile("EXT-X-STREAM-INF:*")
	bandwidthMatch, _ := regexp.Compile("BANDWIDTH=([0-9]+)")
	codecMatch, _ := regexp.Compile("CODECS=")
	nameMatch, _ := regexp.Compile("NAME=")
	resMatch, _ := regexp.Compile("RESOLUTION=")

	dir, _ := filepath.Split(url)

	directives := strings.Split(text, "#")

	for i := 0; i < len(directives); i++ {
		if streamInfMatch.MatchString(directives[i]) {
			var media Playlist
			media.baseUrl = dir

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

			media.playlistUrl = dir + parts[1]

			out.varients = append(out.varients, media)
		}
	}

	return
}

func main() {
	url := "https://test-streams.mux.dev/x36xhzz/x36xhzz.m3u8"

	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil && resp.StatusCode > 299 {
		panic(err)
	}

	playlist := HLSMasterPlaylistParse(string(body), url)
	fmt.Println("{")
	for i := 0; i < len(playlist.varients); i++ {
		playlist.varients[i].HLSPlaylistParse()
		fmt.Println("\t", playlist.varients[i])
	}
	fmt.Println("}")

	// Resolve first varient
	for i := 0; i < len(playlist.varients[0].segments); i++ {
		playlist.varients[0].segments[i].ResolveData()
		// fmt.Println(playlist.varients[0].segments[i].data)
	}

}
