package main

/*
 * A simple go program to download all the high resolution pictures from your facebook albums.
 *
 * To run this:
 * 1. Go to https://developers.facebook.com/tools/explorer/?method=GET&path=me&version=v2.8
 * 2. Get an Access Token: Get Token > Get User Access Token > Check "user_photos"
 * 3. Paste in the app.
 */

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/kennygrant/sanitize"

	fb "github.com/huandu/facebook"
)

var logger *log.Logger

func main() {
	logger = log.New(os.Stdout, "LOG: ", log.Ldate)

	// Get access token manually from https://developers.facebook.com/tools/explorer/?method=GET&path=me&version=v2.8
	var accessToken string
	fmt.Println("Please go to the following URL and to get the access token")
	fmt.Println("\thttps://developers.facebook.com/tools/explorer/?method=GET&path=me&version=v2.8")
	fmt.Printf("\tGet Token (button) > Get User Access Token > Check 'user_photos'\n\n")
	fmt.Print("Paste the access token: ")
	fmt.Scanln(&accessToken)
	var fbApp = fb.New("", "")
	session := fbApp.Session(accessToken)

	res, err := session.Get("/me", nil)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Println("Albums for", res["name"])

	res, _ = session.Get("/me/albums", nil)
	paging, _ := res.Paging(session)
	for {
		items := paging.Data()
		for _, album := range items {
			// Create directory and process album
			os.Mkdir(album.Get("name").(string), os.ModePerm)
			processAlbum(session, album)
		}
		noMore, _ := paging.Next()
		if noMore {
			break
		}
	}

}

func processAlbum(session *fb.Session, album fb.Result) {
	fmt.Printf("%s: %s\n", album.GetField("id"), album.GetField("name"))
	albumName := album.GetField("name").(string)

	// Get the photos in the album
	res, _ := session.Get(fmt.Sprintf("/%s/photos", album.GetField("id")), fb.Params{
		"fields": "name,images",
	})
	paging, _ := res.Paging(session)

	for {
		items := paging.Data()
		for _, photo := range items {
			// Find the largest image.
			var largest fb.Result
			var lastLargestHeight int64
			var lastLargestWidth int64
			var images []fb.Result
			photo.DecodeField("images", &images)
			for _, imageSpecs := range images {
				height, _ := imageSpecs["height"].(json.Number).Int64()
				width, _ := imageSpecs["width"].(json.Number).Int64()
				// fmt.Printf("\t%d x %d\n", height, width)
				if height > lastLargestHeight {
					lastLargestHeight = height
					largest = imageSpecs
				}
				if width > lastLargestWidth {
					lastLargestWidth = width
					largest = imageSpecs
				}
			}
			// fmt.Printf("Final: %s\n\n", largest)

			// Download the image
			photoSource := largest["source"].(string)
			photoURL, _ := url.Parse(photoSource)
			fmt.Printf("\t\t%s: %s\n",
				photo.GetField("id"), photo.GetField("name"))

			extension := filepath.Ext(photoURL.Path)

			response, err := http.Get(photoSource)
			if err != nil {
				logger.Panic(err)
			}
			defer response.Body.Close()

			var filename string
			if photo.Get("name") == nil {
				filename = photo.GetField("id").(string)
			} else {
				filename = photo.GetField("name").(string)
			}

			if utf8.RuneCountInString(filename) > 100 {
				filename = filename[:100]
			}
			filename = sanitize.BaseName(filename)

			file, err := os.Create(filepath.Join(albumName, filename+extension))
			if err != nil {
				logger.Panic(err)
			}
			_, err = io.Copy(file, response.Body)
			if err != nil {
				logger.Panic(err)
			}
			file.Close()
		}

		noMore, _ := paging.Next()
		if noMore {
			break
		}
	}
}
