package htmlutil

import (
	"strings"

	"golang.org/x/net/html"

	"github.com/basecamp/hey-cli/internal/models"
)

// ExtractAttachments returns the downloadable attachments embedded in a Trix
// HTML body. Only <figure data-trix-attachment="{...}"> elements are recognised
// — these carry url/filename/contentType in a JSON attribute and are how the
// editor emits inline images. True MIME attachments (PDFs, docs) are surfaced
// separately by the outer thread page and parsed by ExtractFileAttachments.
func ExtractAttachments(htmlStr string) []models.Attachment {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return nil
	}
	var out []models.Attachment
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "figure" {
			if att := parseTrixAttachment(n); att != nil && att.URL != "" {
				out = append(out, models.Attachment{
					URL:         att.URL,
					Filename:    att.Filename,
					ContentType: att.ContentType,
				})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
}

// ExtractFileAttachments returns true MIME attachments (PDFs, docs, etc.)
// surfaced on the outer /topics/{id}/entries page. They live inside
// <div class="attachments-browser"> blocks, with one <figure class="attachment">
// per file. The downloadable URL is the href on the inner <a> with a
// `download` attribute (a Rails Active Storage redirect path), and the
// filename comes from the same `download` attribute.
//
// Pass the slice of HTML for a single entry — the function does not know
// about entry boundaries.
func ExtractFileAttachments(htmlStr string) []models.Attachment {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return nil
	}
	var out []models.Attachment
	forEachElementWithClass(doc, "attachments-browser", func(browser *html.Node) {
		forEachElementWithClass(browser, "attachment", func(figure *html.Node) {
			if figure.Type != html.ElementNode || figure.Data != "figure" {
				return
			}
			a := findDownloadAnchor(figure)
			if a == nil {
				return
			}
			href := getAttr(a, "href")
			download := strings.Trim(getAttr(a, "download"), `"`)
			if href == "" || download == "" {
				return
			}
			out = append(out, models.Attachment{
				URL:         href,
				Filename:    download,
				ContentType: contentTypeFromFiletype(getAttr(a, "data-filetype")),
			})
		})
	})
	return out
}

// findDownloadAnchor returns the first <a> descendant that has both a non-empty
// href and a non-empty download attribute — these mark the canonical
// blob-redirect link inside an attachment figure.
func findDownloadAnchor(root *html.Node) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" &&
			getAttr(n, "href") != "" && getAttr(n, "download") != "" {
			found = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return found
}

// contentTypeFromFiletype maps the small `data-filetype` hint that HEY emits
// (e.g. "pdf", "image", "doc") to a best-guess MIME type. Returns "" when no
// confident mapping exists — the listing prints "?" in that case.
func contentTypeFromFiletype(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pdf":
		return "application/pdf"
	case "image":
		return "image"
	case "doc":
		return "application/msword"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "xls":
		return "application/vnd.ms-excel"
	case "xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "zip":
		return "application/zip"
	}
	return ""
}
