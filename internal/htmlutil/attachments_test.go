package htmlutil

import "testing"

func TestExtractAttachmentsTrixFigure(t *testing.T) {
	h := `<p>See attached</p><figure data-trix-attachment='{"url":"/rails/blobs/abc/photo.png","filename":"photo.png","contentType":"image/png"}'></figure>`
	got := ExtractAttachments(h)
	if len(got) != 1 {
		t.Fatalf("got %d attachments, want 1", len(got))
	}
	if got[0].URL != "/rails/blobs/abc/photo.png" {
		t.Errorf("URL = %q, want %q", got[0].URL, "/rails/blobs/abc/photo.png")
	}
	if got[0].Filename != "photo.png" {
		t.Errorf("Filename = %q, want %q", got[0].Filename, "photo.png")
	}
	if got[0].ContentType != "image/png" {
		t.Errorf("ContentType = %q, want %q", got[0].ContentType, "image/png")
	}
}

func TestExtractAttachmentsMultiple(t *testing.T) {
	h := `<figure data-trix-attachment='{"url":"/blobs/1/a.png","filename":"a.png","contentType":"image/png"}'></figure>` +
		`<p>middle</p>` +
		`<figure data-trix-attachment='{"url":"/blobs/2/b.pdf","filename":"b.pdf","contentType":"application/pdf"}'></figure>`
	got := ExtractAttachments(h)
	if len(got) != 2 {
		t.Fatalf("got %d attachments, want 2", len(got))
	}
	if got[0].Filename != "a.png" || got[1].Filename != "b.pdf" {
		t.Errorf("filenames = %q, %q; want a.png, b.pdf", got[0].Filename, got[1].Filename)
	}
}

func TestExtractAttachmentsNone(t *testing.T) {
	if got := ExtractAttachments("<p>plain</p>"); len(got) != 0 {
		t.Errorf("got %d attachments, want 0", len(got))
	}
}

func TestExtractAttachmentsMissingURL(t *testing.T) {
	h := `<figure data-trix-attachment='{"filename":"orphan.png","contentType":"image/png"}'></figure>`
	if got := ExtractAttachments(h); len(got) != 0 {
		t.Errorf("got %d attachments, want 0 (URL is required)", len(got))
	}
}

func TestExtractAttachmentsMissingFilename(t *testing.T) {
	h := `<figure data-trix-attachment='{"url":"/blobs/x/file","contentType":"image/png"}'></figure>`
	if got := ExtractAttachments(h); len(got) != 0 {
		t.Errorf("got %d attachments, want 0 (filename is required)", len(got))
	}
}

func TestExtractAttachmentsSkipsActionTextAttachment(t *testing.T) {
	// v1 only handles Trix figures. action-text-attachment markers are skipped
	// because we don't have a confirmed URL attribute path for them yet.
	h := `<action-text-attachment filename="photo.png"><img src="url"></action-text-attachment>`
	if got := ExtractAttachments(h); len(got) != 0 {
		t.Errorf("got %d attachments, want 0 (action-text-attachment unsupported in v1)", len(got))
	}
}

func TestExtractAttachmentsMalformedJSON(t *testing.T) {
	h := `<figure data-trix-attachment='not json'></figure>`
	if got := ExtractAttachments(h); len(got) != 0 {
		t.Errorf("got %d attachments, want 0 for malformed JSON", len(got))
	}
}

func TestExtractFileAttachmentsRealMarkup(t *testing.T) {
	// Captured from a real /topics/{id}/entries response (Slack invoice email).
	// The figure lives inside <div class="attachments-browser"> and the
	// downloadable link is the <a> with both download and href attributes.
	h := `
<div class="attachments-browser attachments-browser--standalone">
  <figure class="attachment attachment--pdf attachment--preview">
    <a id="blob_link_entry_attachment_244811176" class="undecorated"
       download="slack_invoice_SBIE-11510218.pdf"
       data-filetype="pdf"
       data-filename="&quot;slack_invoice_SBIE-11510218.pdf&quot;"
       href="/rails/active_storage/blobs/redirect/abc/slack_invoice_SBIE-11510218.pdf">
      <div class="attachment__frame"><img src="/rails/active_storage/representations/redirect/abc/x/slack_invoice_SBIE-11510218.pdf"></div>
    </a>
    <figcaption class="attachment__caption">
      <a class="undecorated" download="slack_invoice_SBIE-11510218.pdf"
         href="/rails/active_storage/blobs/redirect/abc/slack_invoice_SBIE-11510218.pdf">
        <span class="attachment__name">slack_invoice_SBIE-11510218.pdf</span>
      </a>
    </figcaption>
  </figure>
</div>`
	got := ExtractFileAttachments(h)
	if len(got) != 1 {
		t.Fatalf("got %d attachments, want 1", len(got))
	}
	if got[0].Filename != "slack_invoice_SBIE-11510218.pdf" {
		t.Errorf("filename = %q, want slack_invoice_SBIE-11510218.pdf", got[0].Filename)
	}
	if got[0].URL != "/rails/active_storage/blobs/redirect/abc/slack_invoice_SBIE-11510218.pdf" {
		t.Errorf("URL = %q", got[0].URL)
	}
	if got[0].ContentType != "application/pdf" {
		t.Errorf("content type = %q, want application/pdf", got[0].ContentType)
	}
}

func TestExtractFileAttachmentsMultiple(t *testing.T) {
	h := `
<div class="attachments-browser">
  <figure class="attachment attachment--pdf">
    <a download="invoice.pdf" data-filetype="pdf" href="/rails/active_storage/blobs/redirect/a/invoice.pdf"></a>
  </figure>
  <figure class="attachment attachment--doc">
    <a download="contract.docx" data-filetype="docx" href="/rails/active_storage/blobs/redirect/b/contract.docx"></a>
  </figure>
</div>`
	got := ExtractFileAttachments(h)
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
	if got[0].Filename != "invoice.pdf" || got[1].Filename != "contract.docx" {
		t.Errorf("filenames = %q, %q", got[0].Filename, got[1].Filename)
	}
}

func TestExtractFileAttachmentsNoBrowser(t *testing.T) {
	// A figure outside an attachments-browser must not be picked up
	// (otherwise inline Trix figures would double-count).
	h := `<figure class="attachment attachment--pdf"><a download="x.pdf" href="/x.pdf"></a></figure>`
	if got := ExtractFileAttachments(h); len(got) != 0 {
		t.Errorf("got %d, want 0 (figure outside attachments-browser)", len(got))
	}
}

func TestExtractFileAttachmentsSkipMissingHref(t *testing.T) {
	h := `
<div class="attachments-browser">
  <figure class="attachment"><a download="lonely.pdf"></a></figure>
</div>`
	if got := ExtractFileAttachments(h); len(got) != 0 {
		t.Errorf("got %d, want 0 (no href)", len(got))
	}
}

func TestExtractFileAttachmentsSkipMissingDownload(t *testing.T) {
	h := `
<div class="attachments-browser">
  <figure class="attachment"><a href="/x.pdf"></a></figure>
</div>`
	if got := ExtractFileAttachments(h); len(got) != 0 {
		t.Errorf("got %d, want 0 (no download attribute)", len(got))
	}
}
