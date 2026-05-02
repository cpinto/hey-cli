package htmlutil

import (
	"os"
	"reflect"
	"testing"
)

func TestParseTopicAddressedRealSample(t *testing.T) {
	data, err := os.ReadFile("testdata/topic_sample.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	got := ParseTopicAddressed(string(data))

	wantTo := []string{"alice@example.com", "accounts@example.com"}
	wantCC := []string{"bob@example.com", "chris@example.com", "dana@example.com"}

	if !reflect.DeepEqual(got.To, wantTo) {
		t.Errorf("To = %v, want %v", got.To, wantTo)
	}
	if !reflect.DeepEqual(got.CC, wantCC) {
		t.Errorf("CC = %v, want %v", got.CC, wantCC)
	}
	if len(got.BCC) != 0 {
		t.Errorf("BCC = %v, want empty", got.BCC)
	}
}

func TestParseTopicAddressedMultiRecipientTo(t *testing.T) {
	// Regression: the old regex used a non-greedy `</span>` terminator
	// that truncated at the first inner `</span>`. This fixture embeds
	// inner spans inside each recipient so the old code would have captured
	// only the first address.
	htmlStr := `
<span class="entry__full-recipients">
  <a title="alice@example.com"><span>Alice</span></a>,
  <a title="bob@example.com"><span>Bob</span></a>,
  <a title="carol@example.com"><span>Carol</span></a>
</span>`

	got := ParseTopicAddressed(htmlStr)

	want := []string{"alice@example.com", "bob@example.com", "carol@example.com"}
	if !reflect.DeepEqual(got.To, want) {
		t.Errorf("To = %v, want %v", got.To, want)
	}
}

func TestParseTopicAddressedWithCCAndBCC(t *testing.T) {
	htmlStr := `
<span class="entry__full-recipients">
  <a title="alice@example.com">Alice</a>,
  CC: <a title="bob@example.com">Bob</a>,
  BCC: <a title="carol@example.com">Carol</a>
</span>`

	got := ParseTopicAddressed(htmlStr)

	if !reflect.DeepEqual(got.To, []string{"alice@example.com"}) {
		t.Errorf("To = %v", got.To)
	}
	if !reflect.DeepEqual(got.CC, []string{"bob@example.com"}) {
		t.Errorf("CC = %v", got.CC)
	}
	if !reflect.DeepEqual(got.BCC, []string{"carol@example.com"}) {
		t.Errorf("BCC = %v", got.BCC)
	}
}

func TestParseTopicAddressedBCCOnly(t *testing.T) {
	// "BCC:" must be checked before "CC:" (which is a suffix) — otherwise
	// the BCC bucket never fires.
	htmlStr := `
<span class="entry__full-recipients">
  <a title="alice@example.com">Alice</a>,
  BCC: <a title="carol@example.com">Carol</a>
</span>`

	got := ParseTopicAddressed(htmlStr)

	if !reflect.DeepEqual(got.To, []string{"alice@example.com"}) {
		t.Errorf("To = %v", got.To)
	}
	if len(got.CC) != 0 {
		t.Errorf("CC = %v, want empty", got.CC)
	}
	if !reflect.DeepEqual(got.BCC, []string{"carol@example.com"}) {
		t.Errorf("BCC = %v", got.BCC)
	}
}

func TestParseTopicAddressedUnionsAcrossEntries(t *testing.T) {
	// Regression: the old regex used FindStringSubmatch (first match only),
	// so in a multi-entry thread it silently dropped recipients added in
	// later replies. Union behavior must include everyone.
	htmlStr := `
<div id="entry_1">
  <span class="entry__full-recipients">
    <a title="alice@example.com">Alice</a>,
    CC: <a title="bob@example.com">Bob</a>
  </span>
</div>
<div id="entry_2">
  <span class="entry__full-recipients">
    <a title="alice@example.com">Alice</a>,
    <a title="bob@example.com">Bob</a>,
    CC: <a title="carol@example.com">Carol</a>
  </span>
</div>`

	got := ParseTopicAddressed(htmlStr)

	wantTo := []string{"alice@example.com", "bob@example.com"}
	wantCC := []string{"bob@example.com", "carol@example.com"}

	// Note: bob appears in entry_1 CC and entry_2 To. Dedup is per-bucket,
	// so bob ends up in both — reply logic will still reach him via To.
	if !reflect.DeepEqual(got.To, wantTo) {
		t.Errorf("To = %v, want %v", got.To, wantTo)
	}
	if !reflect.DeepEqual(got.CC, wantCC) {
		t.Errorf("CC = %v, want %v", got.CC, wantCC)
	}
}

func TestParseTopicAddressedEmpty(t *testing.T) {
	got := ParseTopicAddressed("")
	if len(got.To) != 0 || len(got.CC) != 0 || len(got.BCC) != 0 {
		t.Errorf("expected empty, got %+v", got)
	}
}

func TestParseTopicAddressedNoMatch(t *testing.T) {
	got := ParseTopicAddressed(`<div><p>no recipients here</p></div>`)
	if len(got.To) != 0 || len(got.CC) != 0 || len(got.BCC) != 0 {
		t.Errorf("expected empty, got %+v", got)
	}
}

func TestParseTopicAddressedIgnoresNonEmailTitles(t *testing.T) {
	// Titles without `@` (e.g. tooltips like title="Click to expand") must
	// not be treated as addresses.
	htmlStr := `
<span class="entry__full-recipients">
  <button title="Click to expand">...</button>
  <a title="alice@example.com">Alice</a>
</span>`

	got := ParseTopicAddressed(htmlStr)

	if !reflect.DeepEqual(got.To, []string{"alice@example.com"}) {
		t.Errorf("To = %v, want [alice@example.com]", got.To)
	}
}

func TestParseTopicAddressedSupportsSpanAsRecipient(t *testing.T) {
	// Tolerance check: HEY could switch the recipient element from <a>
	// to <span> (or anything else). Attribute-based extraction should
	// still work.
	htmlStr := `
<span class="entry__full-recipients">
  <span title="alice@example.com">Alice</span>,
  <span title="bob@example.com">Bob</span>
</span>`

	got := ParseTopicAddressed(htmlStr)

	want := []string{"alice@example.com", "bob@example.com"}
	if !reflect.DeepEqual(got.To, want) {
		t.Errorf("To = %v, want %v", got.To, want)
	}
}

func TestParseTopicEntriesHTMLAttachments(t *testing.T) {
	// Entry bodies are embedded as HTML-entity-encoded strings inside an iframe
	// srcdoc attribute, marked by the trix-content class. ExtractAttachments
	// runs against the decoded body and should populate Entry.Attachments per
	// entry. Entry 1 has one attachment; entry 2 has none.
	body1 := `&lt;div class=&quot;trix-content&quot;&gt;` +
		`&lt;p&gt;Hello&lt;/p&gt;` +
		`&lt;figure data-trix-attachment=&#39;{&quot;url&quot;:&quot;/rails/blobs/abc/photo.png&quot;,&quot;filename&quot;:&quot;photo.png&quot;,&quot;contentType&quot;:&quot;image/png&quot;}&#39;&gt;&lt;/figure&gt;` +
		`&lt;/div&gt;`
	body2 := `&lt;div class=&quot;trix-content&quot;&gt;&lt;p&gt;No attachments&lt;/p&gt;&lt;/div&gt;`

	htmlStr := `
<article data-entry-id="1" id="entry_1">
  <iframe srcdoc="` + body1 + `"></iframe>
</article>
<article data-entry-id="2" id="entry_2">
  <iframe srcdoc="` + body2 + `"></iframe>
</article>`

	got := ParseTopicEntriesHTML(htmlStr)
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}

	if len(got[0].Attachments) != 1 {
		t.Fatalf("entry 1: got %d attachments, want 1 (body=%q)", len(got[0].Attachments), got[0].BodyHTML)
	}
	att := got[0].Attachments[0]
	if att.Filename != "photo.png" {
		t.Errorf("entry 1 filename = %q, want photo.png", att.Filename)
	}
	if att.URL != "/rails/blobs/abc/photo.png" {
		t.Errorf("entry 1 URL = %q, want /rails/blobs/abc/photo.png", att.URL)
	}
	if att.ContentType != "image/png" {
		t.Errorf("entry 1 content type = %q, want image/png", att.ContentType)
	}

	if len(got[1].Attachments) != 0 {
		t.Errorf("entry 2: got %d attachments, want 0", len(got[1].Attachments))
	}
}

func TestParseTopicEntriesHTMLFileAttachmentsPerEntry(t *testing.T) {
	// File attachments live in <div class="attachments-browser"> blocks inside
	// each entry's <article>. Each entry gets only its own attachments.
	htmlStr := `
<article data-entry-id="1" id="entry_1">
  <message-content data-entry-id="1"></message-content>
  <div class="attachments-browser">
    <figure class="attachment attachment--pdf">
      <a download="invoice-a.pdf" data-filetype="pdf" href="/rails/active_storage/blobs/redirect/a/invoice-a.pdf"></a>
    </figure>
  </div>
</article>
<article data-entry-id="2" id="entry_2">
  <message-content data-entry-id="2"></message-content>
  <div class="attachments-browser">
    <figure class="attachment attachment--doc">
      <a download="contract.docx" data-filetype="docx" href="/rails/active_storage/blobs/redirect/b/contract.docx"></a>
    </figure>
  </div>
</article>`

	got := ParseTopicEntriesHTML(htmlStr)
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if len(got[0].Attachments) != 1 || got[0].Attachments[0].Filename != "invoice-a.pdf" {
		t.Errorf("entry 1 attachments = %+v", got[0].Attachments)
	}
	if len(got[1].Attachments) != 1 || got[1].Attachments[0].Filename != "contract.docx" {
		t.Errorf("entry 2 attachments = %+v", got[1].Attachments)
	}
}

func TestParseTopicEntriesHTMLRecipientsPerEntry(t *testing.T) {
	// Each entry's recipients are scoped to that entry. This is a synthetic
	// fixture mirroring the structure of /topics/{id}/entries output.
	htmlStr := `
<article data-entry-id="1" id="entry_1">
  <span class="entry__full-recipients">
    <a title="alice@example.com">Alice</a>,
    CC: <a title="bob@example.com">Bob</a>
  </span>
</article>
<article data-entry-id="2" id="entry_2">
  <span class="entry__full-recipients">
    <a title="alice@example.com">Alice</a>,
    <a title="bob@example.com">Bob</a>,
    CC: <a title="carol@example.com">Carol</a>
  </span>
</article>`

	got := ParseTopicEntriesHTML(htmlStr)

	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}

	want1 := []string{"alice@example.com", "bob@example.com"}
	want2 := []string{"alice@example.com", "bob@example.com", "carol@example.com"}

	gotEmails := func(entryIdx int) []string {
		var out []string
		for _, c := range got[entryIdx].Recipients {
			out = append(out, c.EmailAddress)
		}
		return out
	}

	if !reflect.DeepEqual(gotEmails(0), want1) {
		t.Errorf("entry 1 recipients = %v, want %v", gotEmails(0), want1)
	}
	if !reflect.DeepEqual(gotEmails(1), want2) {
		t.Errorf("entry 2 recipients = %v, want %v", gotEmails(1), want2)
	}
}
