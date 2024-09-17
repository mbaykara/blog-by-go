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

	// Replace dashes or underscores with spaces
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")

	// Capitalize the first letter of each word
	title = strings.Title(title)

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
			Slug:    slug,
			Date:    fileInfo.ModTime(),
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
	slug := strings.TrimPrefix(r.URL.Path, "/posts/")

	// Load the post's Markdown file based on the slug
	filePath := filepath.Join("posts", slug+".md")
	content, err := RenderMarkdown(filePath)
	if err != nil {
		http.NotFound(w, r) // If the file doesn't exist, show a 404
		return
	}

	// Create a Post object for the post page
	post := PostData{
		Slug:    slug,
		Date:    time.Now(), // Use current time for demonstration, or parse from front matter
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

	// Parse and execute the template with base and post templates
	tmpl := template.Must(template.ParseFiles(
		"templates/base.gohtml",
		"templates/post.gohtml",
	))

	// Execute the base template, which includes the post content block
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
	}
}

func main() {
	http.HandleFunc("/", HomeHandler)
	http.HandleFunc("/posts/", PostHandler) // Route for individual blog posts

	log.Fatal(http.ListenAndServe(":8080", nil))
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
