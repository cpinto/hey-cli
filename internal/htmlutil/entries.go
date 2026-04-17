package htmlutil

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"github.com/basecamp/hey-cli/internal/models"
)

var (
	entryBlockRe  = regexp.MustCompile(`(?s)data-entry-id="(\d+)"`)
	senderRe      = regexp.MustCompile(`id="sender_entry_(\d+)"[^>]*>\s*([^<]+?)\s*<`)
	senderEmailRe = regexp.MustCompile(`(?s)sender_entry_(\d+).*?entry__sender-email[^>]*><span[^>]*>[^<]*</span>([^<]+)<`)
	timeRe        = regexp.MustCompile(`<time[^>]*datetime="([^"]+)"`)
	srcdocRe      = regexp.MustCompile(`(?s)srcdoc="([^"]*trix-content[^"]*)"`)
)

// TopicAddressed holds the To, CC, and BCC recipients for a topic.
type TopicAddressed struct {
	To  []string
	CC  []string
	BCC []string
}

// ParseTopicAddressed extracts To/CC/BCC recipients from a topic's HTML page.
// Unions recipients across every entry__full-recipients block so multi-entry
// threads include people added to CC in later replies. Addresses are read
// from title="…@…" attributes; buckets switch on "CC:" / "BCC:" text nodes
// encountered in document order.
func ParseTopicAddressed(htmlStr string) *TopicAddressed {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return &TopicAddressed{}
	}
	result := &TopicAddressed{}
	toSeen, ccSeen, bccSeen := map[string]bool{}, map[string]bool{}, map[string]bool{}
	forEachElementWithClass(doc, "entry__full-recipients", func(node *html.Node) {
		addr := extractAddressed(node)
		result.To = appendUnique(result.To, addr.To, toSeen)
		result.CC = appendUnique(result.CC, addr.CC, ccSeen)
		result.BCC = appendUnique(result.BCC, addr.BCC, bccSeen)
	})
	return result
}

// extractAddressed walks an entry__full-recipients subtree in document order,
// bucketing each title="…@…" attribute into To/CC/BCC. Bucket switches on
// text nodes containing "BCC:" (checked first because "CC:" is a suffix) or
// "CC:".
func extractAddressed(root *html.Node) *TopicAddressed {
	result := &TopicAddressed{}
	bucket := &result.To
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			switch {
			case strings.Contains(n.Data, "BCC:"):
				bucket = &result.BCC
			case strings.Contains(n.Data, "CC:"):
				bucket = &result.CC
			}
		case html.ElementNode:
			for _, a := range n.Attr {
				if a.Key == "title" {
					v := strings.TrimSpace(a.Val)
					if strings.Contains(v, "@") {
						*bucket = append(*bucket, v)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return result
}

// forEachElementWithClass invokes fn on every element node whose class
// attribute contains the given class token (whitespace-separated).
func forEachElementWithClass(root *html.Node, class string, fn func(*html.Node)) {
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, a := range n.Attr {
				if a.Key == "class" {
					for _, c := range strings.Fields(a.Val) {
						if c == class {
							fn(n)
							break
						}
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
}

func appendUnique(dst, src []string, seen map[string]bool) []string {
	for _, v := range src {
		if seen[v] {
			continue
		}
		seen[v] = true
		dst = append(dst, v)
	}
	return dst
}

// ParseTopicEntriesHTML extracts structured entry data from the HTML page
// served by /topics/{id}/entries. The JSON API does not return full entry
// bodies, so this HTML-based extraction is required.
func ParseTopicEntriesHTML(htmlStr string) []models.Entry {
	// Find unique entry IDs in order
	idMatches := entryBlockRe.FindAllStringSubmatch(htmlStr, -1)
	seen := map[string]bool{}
	var entryIDs []string
	for _, m := range idMatches {
		if !seen[m[1]] {
			seen[m[1]] = true
			entryIDs = append(entryIDs, m[1])
		}
	}

	// Build lookup maps
	senders := map[string]string{}
	for _, m := range senderRe.FindAllStringSubmatch(htmlStr, -1) {
		if _, exists := senders[m[1]]; !exists {
			senders[m[1]] = m[2]
		}
	}
	senderEmails := map[string]string{}
	for _, m := range senderEmailRe.FindAllStringSubmatch(htmlStr, -1) {
		if _, exists := senderEmails[m[1]]; !exists {
			senderEmails[m[1]] = strings.TrimSpace(m[2])
		}
	}

	// Associate times with entries by finding the first <time> after each entry anchor
	entryTimes := map[string]string{}
	for _, eid := range entryIDs {
		anchor := fmt.Sprintf(`id="entry_%s"`, eid)
		idx := strings.Index(htmlStr, anchor)
		if idx < 0 {
			continue
		}
		if m := timeRe.FindStringSubmatch(htmlStr[idx:]); m != nil {
			entryTimes[eid] = m[1]
		}
	}

	// Associate recipients with entries by slicing between entry anchors and
	// running the DOM-based recipient parser on each slice. The flat list
	// unions To/CC/BCC (the entries view doesn't distinguish buckets).
	entryRecipients := map[string][]models.Contact{}
	for i, eid := range entryIDs {
		anchor := fmt.Sprintf(`id="entry_%s"`, eid)
		start := strings.Index(htmlStr, anchor)
		if start < 0 {
			continue
		}
		end := len(htmlStr)
		if i+1 < len(entryIDs) {
			nextAnchor := fmt.Sprintf(`id="entry_%s"`, entryIDs[i+1])
			if n := strings.Index(htmlStr[start:], nextAnchor); n > 0 {
				end = start + n
			}
		}
		addr := ParseTopicAddressed(htmlStr[start:end])
		localSeen := map[string]bool{}
		for _, bucket := range [][]string{addr.To, addr.CC, addr.BCC} {
			for _, a := range bucket {
				if localSeen[a] {
					continue
				}
				localSeen[a] = true
				entryRecipients[eid] = append(entryRecipients[eid], models.Contact{EmailAddress: a})
			}
		}
	}

	// Extract bodies from srcdoc iframes - they appear in entry order
	type body struct{ html, text string }
	bodyMatches := srcdocRe.FindAllStringSubmatch(htmlStr, -1)
	bodies := make([]body, 0, len(bodyMatches))
	for _, m := range bodyMatches {
		raw := m[1]
		raw = strings.ReplaceAll(raw, "&lt;", "<")
		raw = strings.ReplaceAll(raw, "&gt;", ">")
		raw = strings.ReplaceAll(raw, "&quot;", "\"")
		raw = strings.ReplaceAll(raw, "&amp;", "&")
		raw = strings.ReplaceAll(raw, "&#39;", "'")
		bodies = append(bodies, body{html: raw, text: ToText(raw)})
	}

	entries := make([]models.Entry, 0, len(entryIDs))
	for i, eid := range entryIDs {
		id, _ := strconv.ParseInt(eid, 10, 64)
		e := models.Entry{
			ID:        id,
			CreatedAt: entryTimes[eid],
		}
		if name, ok := senders[eid]; ok {
			e.Creator = models.Contact{Name: name, EmailAddress: senderEmails[eid]}
		}
		if recips, ok := entryRecipients[eid]; ok {
			e.Recipients = recips
		}
		if i < len(bodies) {
			e.Body = bodies[i].text
			e.BodyHTML = bodies[i].html
		}
		entries = append(entries, e)
	}

	return entries
}
