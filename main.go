// Package main `README.md` builder script
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
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

func getPkgMetadata(
	wg *sync.WaitGroup,
	result chan<- pkgMetadata,
	name string,
) {
	defer wg.Done()

	resp, err := http.Get(npmsURL + name)
	if err != nil {
		err = errors.Join(
			fmt.Errorf("could not fetch `%s`'s package metadata", name),
			err,
		)
		log.Println(err)
		return
	}
	defer func() {
		if errBody := resp.Body.Close(); errBody != nil {
			log.Fatalln(errBody)
		}
	}()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		err = errors.Join(
			fmt.Errorf("could not read `%s`'s package metadata", name),
			err,
		)
		log.Println(err)
		return
	}

	pkgMetaRaw := pkgMetadataRaw{}
	if err := json.Unmarshal(rawBody, &pkgMetaRaw); err != nil {
		err = errors.Join(
			fmt.Errorf("could not parse `%s`'s package metadata", name),
			err,
		)
		log.Println(err)
		return
	}

	meta := pkgMetadata{}

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

	result <- meta
}

func main() {
	tmpl := template.New(path.Base(readmeTmplPath)).Funcs(tmplFuncMap)
	tmpl, err := tmpl.ParseFiles(readmeTmplPath)
	if err != nil {
		log.Fatalln(err)
	}

	wg := &sync.WaitGroup{}
	results := make(chan pkgMetadata)

	for _, pkg := range packages {
		wg.Add(1)
		go getPkgMetadata(wg, results, pkg)
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	overallMetadata := pkgMetadata{}
	for metadata := range results {
		select {
		case <-ctx.Done():
			log.Fatalln(ctx.Err())
		default:
			overallMetadata.DownloadCount += metadata.DownloadCount
			overallMetadata.Quality += metadata.Quality / float32(len(packages)) * 100
			overallMetadata.Coverage += metadata.Coverage / float32(len(packages)) * 100
		}
	}

	f, err := os.Create("README.md")
	if err != nil {
		log.Fatalln(err)
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
