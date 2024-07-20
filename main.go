package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const npmsURL = "https://api.npms.io/v2/package/"

var packages = []string{"http-responder", "pkgplay", "await-fn"}

type (
	pkgMetadataRaw struct {
		Collected *struct {
			Npm *struct {
				Downloads []struct {
					Count *uint `json:"count"`
				} `json:"downloads"`
			} `json:"npm"`
			Source *struct {
				Coverage *float32 `json:"coverage"`
			} `json:"source"`
		} `json:"collected"`
		Score *struct {
			Detail *struct {
				Quality *float32 `json:"quality"`
			} `json:"detail"`
		} `json:"score"`
	}
	pkgMetadata struct {
		DownloadCount uint    `json:"downloadCount"`
		Quality       float32 `json:"quality"`
		Coverage      float32 `json:"coverage"`
	}
)

func getPkgMetadata(name string) (*pkgMetadata, error) {
	resp, err := http.Get(npmsURL + name)
	if err != nil {
		return nil, err
	}
	defer func() {
		if errBody := resp.Body.Close(); errBody != nil {
			log.Fatalln(errBody)
		}
	}()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	pkgMetaRaw := &pkgMetadataRaw{}
	if err := json.Unmarshal(rawBody, pkgMetaRaw); err != nil {
		return nil, err
	}

	meta := &pkgMetadata{}

	if pkgMetaRaw.Collected != nil &&
		pkgMetaRaw.Collected.Npm != nil &&
		pkgMetaRaw.Collected.Npm.Downloads != nil &&
		len(pkgMetaRaw.Collected.Npm.Downloads) > 0 {
		for _, dlObj := range pkgMetaRaw.Collected.Npm.Downloads {
			if dlObj.Count != nil {
				meta.DownloadCount += *dlObj.Count
			}
		}
	}
	if pkgMetaRaw.Score != nil &&
		pkgMetaRaw.Score.Detail != nil &&
		pkgMetaRaw.Score.Detail.Quality != nil {
		meta.Quality = *pkgMetaRaw.Score.Detail.Quality
	}
	if pkgMetaRaw.Collected != nil &&
		pkgMetaRaw.Collected.Source != nil &&
		pkgMetaRaw.Collected.Source.Coverage != nil {
		meta.Coverage = *pkgMetaRaw.Collected.Source.Coverage
	}

	return meta, nil
}

func main() {
	tmpl, err := template.ParseFiles("./assets/readme-template.md.tmpl")
	if err != nil {
		log.Fatalln(err)
	}

	overallMetadata := &pkgMetadata{}

	for _, pkg := range packages {
		metadata, pkgErr := getPkgMetadata(pkg)
		if pkgErr != nil {
			log.Println(pkgErr)
		}

		overallMetadata.DownloadCount += metadata.DownloadCount
		overallMetadata.Quality += metadata.Quality / float32(len(packages)) * 100
		overallMetadata.Coverage += metadata.Coverage / float32(len(packages)) * 100
	}

	f, err := os.Create("README.md")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if errClose := f.Close(); errClose != nil {
			log.Fatalln(errClose)
		}
	}()

	tmpl.Execute(f, map[string]string{
		"DownloadCount": message.NewPrinter(language.English).
			Sprintf("%d", overallMetadata.DownloadCount),
		"Quality":   fmt.Sprintf("%.2f%%", overallMetadata.Quality),
		"Coverage":  fmt.Sprintf("%.2f%%", overallMetadata.Coverage),
		"TodayDate": time.Now().Format("January 2, 2006"),
	})
}
