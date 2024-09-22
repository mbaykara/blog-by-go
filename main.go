package main

import (
	"bytes"
	"fmt"
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
	files, err := filepath.Glob("post/*.md")
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

// PostHandler serves a specific blog post based on the slug.
func PostHandler(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/post/")

	// Load the post's Markdown file based on the slug
	filePath := filepath.Join("post", slug+".md")
	content, err := RenderMarkdown(filePath)
	if err != nil {
		http.NotFound(w, r) // If the file doesn't exist, show a 404
		return
	}

	// Create a Post object for the post page
	post := PostData{
		Title:   CleanTitle(slug),
		Date:    time.Now(), // Use current time for demonstration, or parse from front matter
		Slug:    slug,
		Content: content,
	}
	// Set up template data
	data := struct {
		Title string
		Post  PostData
	}{
		Title: post.Title,
		Post:  post,
	}
	fmt.Println(data, "data at line 148")

	// Parse and execute the template with base and post templates
	tmpl := template.Must(template.ParseFiles(filepath.Join("templates", "base.gohtml"), filepath.Join("templates", "post.gohtml")))

	// Execute the base template, which includes the post content block
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/", HomeHandler)
	http.HandleFunc("/post/", PostHandler) // Route for individual blog posts
	http.HandleFunc("/about", AboutHandler)
	http.HandleFunc("/contact", ContactHandler)

	log.Fatal(http.ListenAndServe(":8090", nil))
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
	fmt.Println((tmpl), "tmpl at line 185")
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
	fmt.Println(data, "data at line 196")
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
	fmt.Println(tmpl)
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}
