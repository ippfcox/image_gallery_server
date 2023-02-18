package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

type DirInfo struct {
	Name    string
	Path    string
	SubDirs []DirInfo
	Files   []string
}

type FileInfo struct {
	Name     string
	Path     string
	ThumbURL string
	URL      string
	Width    int
	Height   int
}

var dir = flag.String("dir", ".", "root dir")
var port = flag.Int("port", 8000, "listen port")
var absdir string

func main() {
	flag.Parse()
	dirfi, err := os.Lstat(*dir)
	if os.IsNotExist(err) {
		log.Fatalf("<%s> IsNotExist", *dir)
	}
	if !dirfi.IsDir() {
		log.Fatalf("<%s> !IsDir", *dir)
	}
	absdir, _ = filepath.Abs(*dir)
	log.Printf("Serve dir: <%s>", absdir)

	http.HandleFunc("/", dirHandler)
	http.HandleFunc("/dir/", dirHandler)
	http.HandleFunc("/static/", staticHandler)
	http.HandleFunc("/image/", imageHandler)
	http.HandleFunc("/thumb/", thumbHandler)
	log.Printf("Listen and serve on :%d", *port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		log.Fatal("Failed to ListenAndServe: ", err.Error())
	}
}

func dirHandler(w http.ResponseWriter, r *http.Request) {
	root := absdir
	if len(r.URL.Path) > len("/dir/") {
		root = string(r.URL.Path[len("/dir/"):])
	}

	dirInfos := createDirInfos(root)

	t, err := template.ParseFiles("dir.html")
	if err != nil {
		log.Fatal("Parse template error:", err)
	}

	if err := t.Execute(w, dirInfos); err != nil {
		log.Fatal("Execute template error:", err)
	}
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, r.URL.Path[1:])
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/image/"):]
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Failed to open file %s: %v\n", path, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	io.Copy(w, file)
}

func thumbHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/thumb/"):]
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Failed to open file %s: %v\n", path, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	img, err := imaging.Decode(file)
	if err != nil {
		log.Printf("Failed to decode image %s: %v\n", path, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	thumb := imaging.Resize(img, 270, 480, imaging.Lanczos)
	imaging.Encode(w, thumb, imaging.JPEG)
}

func createDirInfos(path string) DirInfo {
	dirInfo := DirInfo{
		Name:    filepath.Base(path),
		Path:    path,
		Files:   make([]string, 0),
		SubDirs: make([]DirInfo, 0),
	}
	fileInfoList, _ := os.ReadDir(path)
	for _, fi := range fileInfoList {
		if fi.IsDir() {
			dirInfo.SubDirs = append(dirInfo.SubDirs, createDirInfos(filepath.Join(path, fi.Name())))
		} else {
			ext := filepath.Ext(fi.Name())
			if ext == ".jpg" || ext == ".png" || ext == ".gif" || ext == ".bmp" {
				dirInfo.Files = append(dirInfo.Files, filepath.Join(path, fi.Name()))
			}
		}
	}
	return dirInfo
}
