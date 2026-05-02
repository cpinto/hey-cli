package smoke_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestAttachmentsList exercises `hey attachments <thread-id>` against any
// thread in the imbox. The dev server may have no threads with attachments;
// in that case the command must succeed and return zero rows. We don't fail
// the test for an empty result.
func TestAttachmentsList(t *testing.T) {
	topicID := pickAnyImboxTopicID(t)
	if topicID == "" {
		t.Skip("no postings in imbox")
	}

	resp := heyJSON(t, "attachments", topicID)
	type Attachment struct {
		URL         string `json:"url"`
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
	}
	type Row struct {
		EntryID     int          `json:"entry_id"`
		From        string       `json:"from"`
		Attachments []Attachment `json:"attachments"`
	}
	rows := dataAs[[]Row](t, resp)
	for _, r := range rows {
		if len(r.Attachments) == 0 {
			t.Errorf("entry %d listed with zero attachments — list should omit empty entries", r.EntryID)
		}
		for _, a := range r.Attachments {
			if a.Filename == "" {
				t.Errorf("entry %d has attachment with empty filename", r.EntryID)
			}
			if a.URL == "" {
				t.Errorf("entry %d attachment %s has empty URL", r.EntryID, a.Filename)
			}
		}
	}
}

// TestAttachmentsDownload walks the imbox looking for any thread with at
// least one attachment, then downloads it to a temp dir. Skips gracefully
// when no thread in the imbox carries attachments.
func TestAttachmentsDownload(t *testing.T) {
	type Posting struct {
		AppURL              string `json:"app_url"`
		IncludesAttachments bool   `json:"includes_attachments"`
	}
	type BoxResp struct {
		Postings []Posting `json:"postings"`
	}

	resp := heyJSON(t, "box", "imbox")
	data := dataAs[BoxResp](t, resp)

	var topicID string
	for _, p := range data.Postings {
		if !p.IncludesAttachments {
			continue
		}
		id := extractTopicID(p.AppURL)
		if id == "" {
			continue
		}
		// Confirm via the attachments listing — `includes_attachments` is set
		// by the server; the parser only sees Trix figures, so the two can
		// disagree (e.g. action-text-attachment-only threads).
		listing := heyJSON(t, "attachments", id)
		type Row struct {
			EntryID int `json:"entry_id"`
		}
		rows := dataAs[[]Row](t, listing)
		if len(rows) > 0 {
			topicID = id
			break
		}
	}
	if topicID == "" {
		t.Skip("no imbox threads with downloadable Trix attachments")
	}

	out := t.TempDir()
	dlResp := heyJSON(t, "attachments", "download", topicID, "--output", out)
	type Result struct {
		Path     string `json:"path"`
		Filename string `json:"filename"`
		Bytes    int    `json:"bytes"`
	}
	results := dataAs[[]Result](t, dlResp)
	if len(results) == 0 {
		t.Fatal("download returned zero results despite listing showing attachments")
	}
	for _, r := range results {
		if r.Bytes <= 0 {
			t.Errorf("downloaded %s is empty", r.Filename)
		}
		info, err := os.Stat(r.Path)
		if err != nil {
			t.Errorf("stat %s: %v", r.Path, err)
			continue
		}
		if info.Size() <= 0 {
			t.Errorf("file %s on disk is empty", r.Path)
		}
		if filepath.Dir(r.Path) != out {
			t.Errorf("file %s landed outside %s", r.Path, out)
		}
	}
}

// TestAttachmentsDownloadIndexRequiresEntry verifies that --index without
// --entry produces a usage error rather than silently picking an attachment.
func TestAttachmentsDownloadIndexRequiresEntry(t *testing.T) {
	topicID := pickAnyImboxTopicID(t)
	if topicID == "" {
		t.Skip("no postings in imbox")
	}
	_, stderr := heyFail(t, "attachments", "download", topicID, "--index", "1")
	assertContains(t, stderr, "--index requires --entry")
}

func pickAnyImboxTopicID(t *testing.T) string {
	t.Helper()
	type Posting struct {
		AppURL string `json:"app_url"`
	}
	type BoxResp struct {
		Postings []Posting `json:"postings"`
	}
	resp := heyJSON(t, "box", "imbox")
	data := dataAs[BoxResp](t, resp)
	if len(data.Postings) == 0 {
		return ""
	}
	id := extractTopicID(data.Postings[0].AppURL)
	if id == "" {
		t.Fatal(fmt.Sprintf("could not extract topic ID from %q", data.Postings[0].AppURL))
	}
	return id
}
