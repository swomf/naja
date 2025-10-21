package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Video struct {
	ID       string // for html
	BaseName string
	Versions []string // e.g. demo.mp4, demo_edit.mp4, demo-edit2.mp4 (grouped on '_' and '-')
}

func main() {
	videos, err := collectVideos("video")
	if err != nil {
		log.Fatal(err)
	}

	tmpl, err := template.ParseFiles("web/index.html")
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/video/", http.StripPrefix("/video/", http.FileServer(http.Dir("video"))))
	http.Handle("/thumbnails/", http.StripPrefix("/thumbnails/", http.FileServer(http.Dir("thumbnails"))))
	http.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("web"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := tmpl.Execute(w, map[string]any{"Videos": videos})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	log.Println("serving at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// helper for grouping versions in collectVideos
func splitBase(name string) string {
	cut := len(name)
	if i := strings.Index(name, "_"); i != -1 && i < cut {
		cut = i
	}
	if i := strings.Index(name, "-"); i != -1 && i < cut {
		cut = i
	}
	return name[:cut]
}

func collectVideos(dir string) ([]Video, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	versionCollection := make(map[string][]string) // f: string mapsto list of strings
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if !strings.HasSuffix(name, ".mp4") {
			continue
		}
		base := strings.TrimSuffix(name, ".mp4")
		baseMain := splitBase(base)
		versionCollection[baseMain] = append(versionCollection[baseMain], name)
	}

	var videos []Video
	keys := make([]string, 0, len(versionCollection))
	for k := range versionCollection {
		keys = append(keys, k) // sort on baseMain name
	}
	sort.Strings(keys)

	for i, base := range keys {
		versions := versionCollection[base]
		sort.Strings(versions) // sort the video versions in a collection

		videos = append(videos, Video{
			ID:       "vid" + strconv.Itoa(i+1),
			BaseName: base,
			Versions: versions,
		})
	}

	return videos, nil
}
