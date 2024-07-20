// Package main `README.md` builder script
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"text/template"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	npmsURL        = "https://api.npms.io/v2/package/"
	readmeTmplPath = "./assets/readme-template.md.tmpl"
)

var (
	packages    = []string{"http-responder", "pkgplay", "await-fn"}
	tmplFuncMap = map[string]any{
		"formatNumber": func(n uint) string {
			return message.NewPrinter(language.English).
				Sprintf("%d", n)
		},
		"formatPercent": func(n float32) string {
			return fmt.Sprintf("%.2f%%", n)
		},
		"todayDate": func() string {
			return time.Now().Format("January 2, 2006")
		},
	}
)

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
	tmpl := template.New(path.Base(readmeTmplPath)).Funcs(tmplFuncMap)
	tmpl, err := tmpl.ParseFiles(readmeTmplPath)
	if err != nil {
		log.Fatalln(err)
	}

	overallMetadata := &pkgMetadata{}

	for _, pkg := range packages {
		metadata, errMetadata := getPkgMetadata(pkg)
		if errMetadata != nil {
			log.Println(errMetadata)
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

	if err := tmpl.Execute(f, overallMetadata); err != nil {
		log.Fatalln(err)
	}
}
