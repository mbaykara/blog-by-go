package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Post represents a blog post with a title, date, and content.
type PostData struct {
	Title   string
	Date    time.Time
	Slug    string
	Content template.HTML // Content after converting from Markdown
}

// TemplateData holds the data passed to the template.
type TemplateData struct {
	Title string
	Posts []PostData
}

// RenderMarkdown converts Markdown content to HTML.
func RenderMarkdown(filePath string) (template.HTML, error) {
	md, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	markdown := goldmark.New(
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"),
			),
		),
	)
	remainingMd, err := frontmatter.Parse(strings.NewReader(string(md)), &md)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	err = markdown.Convert([]byte(remainingMd), &buf)
	if err != nil {
		panic(err)
	}
	return template.HTML(buf.String()), nil
}
func CleanTitle(filename string) string {
	// Remove the extension (.md) if present
	title := strings.TrimSuffix(filename, filepath.Ext(filename))

	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")

	// Capitalize the first letter of each word
	title = cases.Title(language.English).String(title)

	return title
}

// LoadBlogPosts loads the blog posts from Markdown files and sorts them by date.
func LoadBlogPosts() ([]PostData, error) {
	var posts []PostData

	// Get all markdown files from the "content" directory
	files, err := filepath.Glob("posts/*.md")
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		// Extract the filename without the extension to use as the Title and Slug
		filename := filepath.Base(file)
		slug := strings.TrimSuffix(filename, filepath.Ext(filename))
		title := CleanTitle(filename)

		// Use the file's ModTime as the post's Date
		fileInfo, err := os.Stat(file)
		if err != nil {
			return nil, err
		}

		// Load and convert the Markdown content to HTML
		content, err := RenderMarkdown(file)
		if err != nil {
			return nil, err
		}

		// Create a Post object
		post := PostData{
			Title:   title,
			Date:    fileInfo.ModTime(),
			Slug:    slug,
			Content: content,
		}
		posts = append(posts, post)
	}

	// Sort posts by date (latest first)
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})

	return posts, nil
}

func main() {
	http.HandleFunc("/", HomeHandler)
	http.HandleFunc("/about", AboutHandler)
	http.HandleFunc("/contact", ContactHandler)
	http.HandleFunc("/post/", PostHandler)

	log.Fatal(http.ListenAndServe(":8090", nil))
}

func PostHandler(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Path[len("/post/"):]
	post, err := LoadPost(slug)
	if err != nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}
	// data := struct {
	// 	Title string
	// 	Post  PostData
	// }{
	// 	Title: post.Title,
	// 	Post:  post,
	// }
	// fmt.Println(data)

	tmpl := template.Must(template.ParseFiles(filepath.Join("templates", "base.gohtml"), filepath.Join("templates", "post.gohtml")))
	if err := tmpl.Execute(w, post); err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

func LoadPost(slug string) (PostData, error) {
	// Find the Markdown file with the given slug
	file := filepath.Join("posts", slug+".md")
	if _,
		err := os.Stat(file); os.IsNotExist(err) {
		return PostData{}, err
	}

	// Load and convert the Markdown content to HTML
	content, err := RenderMarkdown(file)
	if err != nil {
		return PostData{}, err
	}

	// Extract the filename without the extension to use as the Title
	filename := filepath.Base(file)
	title := CleanTitle(filename)

	// Use the file's ModTime as the post's Date
	fileInfo, err := os.Stat(file)
	if err != nil {
		return PostData{}, err
	}

	// Create a Post object
	post := PostData{
		Title:   title,
		Date:    fileInfo.ModTime(),
		Slug:    slug,
		Content: content,
	}

	return post, nil
}

// HomeHandler renders the home page with blog posts.
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	posts, err := LoadBlogPosts()
	if err != nil {
		log.Fatal(err)
	}

	data := TemplateData{
		Title: "My Blog",
		Posts: posts,
	}

	tmpl := template.Must(template.ParseFiles(filepath.Join("templates", "base.gohtml"), filepath.Join("templates", "home.gohtml")))
	if err := tmpl.Execute(w, data); err != nil {
		log.Fatal(err)
	}
}

// AboutHandler serves the About page.
// AboutHandler serves the About page.
func AboutHandler(w http.ResponseWriter, r *http.Request) {
	content, err := RenderMarkdown("nav/about.md")
	if err != nil {
		http.Error(w, "Error loading about page", http.StatusInternalServerError)
		return
	}
	data := struct {
		Title   string
		Content template.HTML
	}{
		Title:   "About Me",
		Content: content,
	}
	tmpl := template.Must(template.ParseFiles(filepath.Join("templates", "base.gohtml"), filepath.Join("templates", "about.gohtml")))
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

func ContactHandler(w http.ResponseWriter, r *http.Request) {
	content, err := RenderMarkdown("nav/contact.md")
	if err != nil {
		http.Error(w, "Error loading contact page", http.StatusInternalServerError)
		return
	}
	data := struct {
		Title   string
		Content template.HTML
	}{
		Title:   "Contact Me",
		Content: content,
	}
	tmpl := template.Must(template.ParseFiles(filepath.Join("templates", "base.gohtml"), filepath.Join("templates", "contact.gohtml")))
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}
