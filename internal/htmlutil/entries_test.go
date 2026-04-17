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
