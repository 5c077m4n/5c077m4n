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

type pkgMetadataRaw struct {
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
type pkgMetadata struct {
	DownloadCount uint    `json:"downloadCount"`
	Quality       float32 `json:"quality"`
	Coverage      float32 `json:"coverage"`
}

func getPkgMetadata(name string) (*pkgMetadata, error) {
	resp, err := http.Get(npmsURL + name)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	pkgMetaRaw := &pkgMetadataRaw{}
	if err := json.Unmarshal(rawBody, pkgMetaRaw); err != nil {
		return nil, err
	}

	var dlCount uint
	if pkgMetaRaw.Collected != nil &&
		pkgMetaRaw.Collected.Npm != nil &&
		pkgMetaRaw.Collected.Npm.Downloads != nil &&
		len(pkgMetaRaw.Collected.Npm.Downloads) > 0 {
		for _, dlObj := range pkgMetaRaw.Collected.Npm.Downloads {
			if dlObj.Count != nil {
				dlCount += *dlObj.Count
			}
		}
	}
	var quality float32
	if pkgMetaRaw.Score != nil &&
		pkgMetaRaw.Score.Detail != nil &&
		pkgMetaRaw.Score.Detail.Quality != nil {
		quality = *pkgMetaRaw.Score.Detail.Quality
	}
	var coverage float32
	if pkgMetaRaw.Collected != nil &&
		pkgMetaRaw.Collected.Source != nil &&
		pkgMetaRaw.Collected.Source.Coverage != nil {
		coverage = *pkgMetaRaw.Collected.Source.Coverage
	}

	return &pkgMetadata{
		DownloadCount: dlCount,
		Quality:       quality,
		Coverage:      coverage,
	}, nil
}

func main() {
	tmpl, err := template.ParseFiles("./assets/readme-template.md.tmpl")
	if err != nil {
		log.Fatalln(err)
	}

	overallMetadata := &pkgMetadata{}

	packages := []string{"http-responder", "pkgplay", "await-fn"}
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
		if err := f.Close(); err != nil {
			log.Fatalln(err)
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
