package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
)

type P5File struct {
	Name           string   `json:"name"`
	Content        string   `json:"content"`
	Id             string   `json:"id"`
	UpdatedAt      string   `json:"updatedAt"`
	CreatedAt      string   `json:"createdAt"`
	FileType       string   `json:"fileType"`
	Children       []string `json:"children"`
	IsSelectedFile bool     `json:"isSelectedFile"`
	Url            string   `json:"url"`
}

type Project struct {
	Id_       string   `json:"_id"`
	Id        string   `json:"id"`
	Name      string   `json:"name"`
	UpdatedAt string   `json:"updatedAt"`
	CreatedAt string   `json:"createdAt"`
	Files     []P5File `json:"files"`
}

func fetch_user_json(username string) []byte {

	url := fmt.Sprintf("https://editor.p5js.org/editor/%s/projects", username)

	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Printf("StatusCode: %d\n", res.StatusCode)

	if res.StatusCode != 200 {
		fmt.Printf("%s\n", body)
		log.Fatal(res.StatusCode)
	}

	filename := fmt.Sprintf("%s.json", username)
	//fmt.Printf("%s", body)
	err = os.WriteFile(filename, body, 0644)
	if err != nil {
		log.Fatal(err)
	}
	return body
}

func create_file_map(project Project) map[string]P5File {
	var m map[string]P5File
	m = make(map[string]P5File)
	for _, file := range project.Files {
		m[file.Id] = file
	}
	return m
}

func get_file(project Project, fileid string) P5File {
	var f P5File
	for _, file := range project.Files {
		if file.Id == fileid {
			f = file
		}
	}
	return f
}

func get_root(proj Project) P5File {
	var root P5File
	for _, file := range proj.Files {
		if file.FileType == "folder" && file.Name == "root" {
			root = file
			break
		}
	}
	return root
}

func create_project_files(username string, project Project) {
	file_map := create_file_map(project)
	project_root := fmt.Sprintf("%s/%s/%s", "output", username, project.Id)
	//fmt.Printf("creating project folder: %s\n", project_root)
	err := os.MkdirAll(project_root, 0755)
	if err != nil {
		log.Fatal(err)
	}
	root_file := get_root(project)
	create_files(root_file, project_root, file_map)
}

func create_files(file P5File, path string, file_map map[string]P5File) {
	if file.FileType == "folder" {
		var newpath string
		if file.Name == "root" {
			newpath = strings.Clone(path)
		} else {
			newpath = path + "/" + file.Name
		}
		//fmt.Printf("creating folder: %s\n", newpath)
		os.MkdirAll(newpath, 0755)
		for _, child_id := range file.Children {
			create_files(file_map[child_id], newpath, file_map)
		}
		return
	} else {
		filename := path + "/" + file.Name
		//fmt.Printf("would create file: %s\n", filename)
		if len(file.Url) == 0 {
			err := os.WriteFile(filename, []byte(file.Content), 0644)
			if err != nil {
				log.Fatal(err)
			}
		}
		// Download
		if len(file.Url) > 0 {
			fmt.Printf("fetching %s\n --> %s\n", file.Url, file.Name)
			resp, err := http.Get(file.Url)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				fmt.Printf("status: %d\n", resp.StatusCode)
				return
			}
			out, err := os.Create(filename)
			if err != nil {
				log.Fatal(err)
			}
			defer out.Close()

			_, err = io.Copy(out, resp.Body)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func create_index(username string, projects []Project) {
	var (
		html strings.Builder
		str  string
	)

	output_dir := fmt.Sprintf("%s/%s", "output", username)
	html.WriteString("<table>")
	for _, proj := range projects {
		html.WriteString("<tr>")
		str = fmt.Sprintf("<td><a href=\"%s/index.html\" target=_blank>%s</a></td>", proj.Id, proj.Name)
		html.WriteString(str)
		str = fmt.Sprintf("<td><a href=\"https://editor.p5js.org/%s/sketches/%s\" target=_blank>p5 editor</a></td>", username, proj.Id)
		html.WriteString(str)
		html.WriteString("</tr>")
	}
	html.WriteString("</table>")
	out_file := fmt.Sprintf("%s/%s", output_dir, "index.html")
	os.WriteFile(out_file, []byte(html.String()), 0644)
}

var (
	projects []Project
)

func main() {

	username := os.Args[1]

	var (
		err     error
		jsonStr []byte
	)

	filename := fmt.Sprintf("%s.json", username)

	jsonStr, err = os.ReadFile(filename)

	if errors.Is(err, fs.ErrNotExist) {
		fmt.Printf("%s does not exist.. fetching..\n", filename)
		jsonStr = fetch_user_json(username)
	} else if err != nil {
		log.Fatal(err)
		return
	}

	err = json.Unmarshal(jsonStr, &projects)
	if err != nil {
		log.Fatal(err)
	}

	nProject := len(projects)
	for i, proj := range projects {
		fmt.Printf("%3d/%d %s..\n", i+1, nProject, proj.Name)
		create_project_files(username, proj)
	}

	create_index(username, projects)

} // main
