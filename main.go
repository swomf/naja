package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Sometimes a video will have unusual underscore distributions that
// naja cannot reason about. Associate the video title to its intended
// collection in basename_overrides.json.
var overrideBases = map[string]string{}

const overrideBasesPath = "basename_overrides.json"

type Video struct {
	ID       string // for html
	BaseName string
	// versions get grouped into collections based on on '_' and '-'
	//
	// e.g. demo.mp4, demo_edit.mp4, demo-edit2.mp4 -> baseMain is demo
	// or a\_1.mp4, a\_2.mp4 -> baseMain is a
	Versions    []string
	ThumbSource string // mp4 used to generate base name .jpg
}

func main() {
	overrideBases = initBaseNameOverrides()

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
		if err := tmpl.Execute(w, map[string]any{"Videos": videos}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// TODO: perhaps ffmpeg parallelization is too ram-heavy?
	err = generateThumbnails(videos, "video", "thumbnails", runtime.NumCPU())
	if err != nil {
		log.Println("warning: couldnt print thumbnails (is ffmpeg in path?): ", err)
	}

	log.Println("serving at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initBaseNameOverrides() map[string]string {
	cfg := make(map[string]string)
	data, err := os.ReadFile(overrideBasesPath)
	if err != nil {
		log.Printf("warning: config file %q not found, continuing with empty config\n", overrideBasesPath)
		return cfg
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("warning: failed to parse %q: %v; continuing with empty config\n", overrideBasesPath, err)
		return map[string]string{}
	}
	return cfg
}

// helper for grouping versions in collectVideos.
// returns text before the first '-' or '_'
func splitBase(name string) string {
	val, ok := overrideBases[name]
	if ok {
		return val
	}

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

		// for thumbnail source, use base.mp4 or first version in videos
		// prefer base.mp4, otherwise first version
		thumbSource := versions[0]
		baseFile := base + ".mp4"
		for _, v := range versions {
			if v == baseFile {
				thumbSource = v
				break
			}
		}

		videos = append(videos, Video{
			ID:          "vid" + strconv.Itoa(i+1),
			BaseName:    base,
			Versions:    versions,
			ThumbSource: thumbSource,
		})
	}

	return videos, nil
}

func generateThumbnails(videos []Video, videoDir, thumbDir string, jobs int) error {
	if err := os.MkdirAll(thumbDir, 0o755); err != nil {
		return err
	}

	semaphore := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	var firstErr error
	var mu sync.Mutex

	for _, v := range videos {
		out := filepath.Join(thumbDir, v.BaseName+".jpg")
		if _, err := os.Stat(out); err == nil {
			continue // don't regenerate an already existing thumbnail.jpg
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(v Video) {
			defer wg.Done()
			defer func() { <-semaphore }()

			cmd := exec.Command(
				"ffmpeg",
				"-hide_banner", "-loglevel", "error",
				"-ss", "00:00:01",
				"-i", filepath.Join(videoDir, v.ThumbSource),
				"-vframes", "1",
				"-q:v", "3",
				out,
			)

			if err := cmd.Run(); err != nil {
				// make sure firstErr writing isnt ub
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(v)
	}

	wg.Wait()
	return firstErr
}
