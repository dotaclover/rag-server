package main

import (
"embed"
"fmt"
"html/template"
"log"
"net/http"
)

//go:embed templates/*
var content embed.FS

func main() {
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
log.Println("Request received")

// Try to read the file directly first
data, err := content.ReadFile("templates/index.html")
if err != nil {
log.Printf("Read file error: %v", err)
http.Error(w, "File read error: "+err.Error(), 500)
return
}

log.Printf("File read successfully: %d bytes", len(data))

// Try template parsing
tmpl, err := template.ParseFS(content, "templates/index.html")
if err != nil {
log.Printf("Template parse error: %v", err)
http.Error(w, "Template parse error: "+err.Error(), 500)
return
}

log.Println("Template parsed successfully")

// Set headers
w.Header().Set("Content-Type", "text/html; charset=utf-8")

// Execute template
if err := tmpl.Execute(w, nil); err != nil {
log.Printf("Template execute error: %v", err)
fmt.Fprintf(w, "Template execute error: %v", err)
return
}

log.Println("Template executed successfully")
})

log.Println("Test server on :9093")
log.Fatal(http.ListenAndServe(":9093", nil))
}
