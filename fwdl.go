package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	instanceUrl, path := parseUrl(os.Args[1])
	albumId := getAlbumId(path)
	albumDir := createAlbumDir(instanceUrl, albumId)
	receiveAlbum(instanceUrl, albumDir, albumId)

	os.Exit(0)
}

func receiveAlbum(instanceUrl, pathToSave, id string) {
	tracksUrl := instanceUrl + "/api/v1/tracks?album=" + id
	rawBody := getWithBody(tracksUrl)

	var jsonMap map[string]any
	json.Unmarshal(rawBody, &jsonMap)
	albumTracks := jsonMap["results"].([]any)

	var wg sync.WaitGroup
	for _, obj := range albumTracks {
		item := obj.(map[string]any)
		listenUrl := item["listen_url"].(string)
		title := item["title"].(string)
		uploads := item["uploads"].([]any)
		if len(uploads) < 1 {
			die("%v: empty uploads for %q", os.Args[0], title)
		}
		ext := uploads[0].(map[string]any)["extension"].(string)

		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Printf("Downloadng %v\n", title)
			trackUrl := instanceUrl + listenUrl
			rawBody := getWithBody(trackUrl)
			fpath := filepath.Join(pathToSave, fmt.Sprintf("%s.%s", title, ext))

			err := os.WriteFile(fpath, rawBody, 0644)
			check(err)
			fmt.Printf("Done %v\n", title)
		}()
	}

	wg.Wait()
}

func createAlbumDir(instanceUrl, id string) string {
	albumUrl := instanceUrl + "/api/v1/albums/" + id
	rawBody := getWithBody(albumUrl)

	var jsonMap map[string]any
	json.Unmarshal(rawBody, &jsonMap)
	dirPath := jsonMap["title"].(string)

	err := os.Mkdir(dirPath, os.ModePerm)
	check(err)

	return dirPath
}

func getWithBody(url string) []byte {
	res, err := http.Get(url)
	check(err)
	defer res.Body.Close()

	rawBody, err := io.ReadAll(res.Body)
	check(err)

	return rawBody
}

func getAlbumId(urlPath string) string {
	if urlPath[len(urlPath)-1] == '/' {
		urlPath = urlPath[:len(urlPath)-1]
	}

	id := path.Base(urlPath)
	if id == "" {
		die("%v: couldn't get album id from path %q\n", os.Args[0], urlPath)
	}

	return id
}

func parseUrl(urlToParse string) (string, string) {
	parsed, err := url.Parse(urlToParse)
	check(err)
	if !strings.Contains(parsed.Path, "album") {
		die("%v: incorrect url %q\n", os.Args[0], parsed.Path)
	}

	hostWithScheme := parsed.Scheme + "://" + parsed.Host
	return hostWithScheme, parsed.Path
}

func check(err error) {
	if err != nil {
		die("%v: %v\n", os.Args[0], err)
	}
}

func usage() {
	die("usage: %v URL\n", os.Args[0])
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
