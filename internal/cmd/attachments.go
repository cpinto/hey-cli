package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/basecamp/hey-cli/internal/htmlutil"
	"github.com/basecamp/hey-cli/internal/models"
	"github.com/basecamp/hey-cli/internal/output"
)

type attachmentsCommand struct {
	cmd *cobra.Command
}

func newAttachmentsCommand() *attachmentsCommand {
	c := &attachmentsCommand{}
	c.cmd = &cobra.Command{
		Use:   "attachments <thread-id>",
		Short: "List attachments in an email thread",
		Annotations: map[string]string{
			"agent_notes": "Lists attachments per entry. Use the download subcommand to save files locally; --entry and --index narrow the selection.",
		},
		Example: `  hey attachments 12345
  hey attachments 12345 --json
  hey attachments download 12345 --output ~/Downloads`,
		RunE: c.run,
		Args: usageExactOneArg(),
	}
	c.cmd.AddCommand(newAttachmentsDownloadCommand().cmd)
	return c
}

func (c *attachmentsCommand) run(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}
	threadID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return output.ErrUsage(fmt.Sprintf("invalid thread ID: %s", args[0]))
	}

	entries, err := fetchThreadEntries(cmd.Context(), threadID)
	if err != nil {
		return err
	}

	type entryListing struct {
		EntryID     int64               `json:"entry_id"`
		From        string              `json:"from,omitempty"`
		CreatedAt   string              `json:"created_at,omitempty"`
		Attachments []models.Attachment `json:"attachments"`
	}
	rows := make([]entryListing, 0, len(entries))
	total := 0
	for _, e := range entries {
		if len(e.Attachments) == 0 {
			continue
		}
		rows = append(rows, entryListing{
			EntryID:     e.ID,
			From:        senderDisplay(e),
			CreatedAt:   e.CreatedAt,
			Attachments: e.Attachments,
		})
		total += len(e.Attachments)
	}

	if writer.IsStyled() {
		w := cmd.OutOrStdout()
		if total == 0 {
			fmt.Fprintf(w, "No attachments in thread %d\n", threadID)
			return nil
		}
		for i, r := range rows {
			if i > 0 {
				fmt.Fprintln(w, strings.Repeat("─", 60))
			}
			fmt.Fprintf(w, "From: %s  [%s]  #%d\n", r.From, shortDate(r.CreatedAt), r.EntryID)
			for j, att := range r.Attachments {
				ct := att.ContentType
				if ct == "" {
					ct = "?"
				}
				fmt.Fprintf(w, "  [%d] %s (%s)\n", j+1, att.Filename, ct)
			}
		}
		return nil
	}

	return writeOK(rows,
		output.WithSummary(fmt.Sprintf("%d attachments across %d entries in thread %d", total, len(rows), threadID)),
		output.WithBreadcrumbs(
			output.Breadcrumb{
				Action:      "download",
				Command:     fmt.Sprintf("hey attachments download %d", threadID),
				Description: "Download attachments from this thread",
			},
		),
	)
}

type attachmentsDownloadCommand struct {
	cmd     *cobra.Command
	entryID int64
	index   int
	outDir  string
}

func newAttachmentsDownloadCommand() *attachmentsDownloadCommand {
	c := &attachmentsDownloadCommand{}
	c.cmd = &cobra.Command{
		Use:   "download <thread-id>",
		Short: "Download attachments from an email thread",
		Example: `  hey attachments download 12345
  hey attachments download 12345 --entry 67890
  hey attachments download 12345 --entry 67890 --index 1
  hey attachments download 12345 --output ~/Downloads`,
		RunE: c.run,
		Args: usageExactOneArg(),
	}
	c.cmd.Flags().Int64Var(&c.entryID, "entry", 0, "Limit to a single entry by entry ID")
	c.cmd.Flags().IntVar(&c.index, "index", 0, "1-based attachment index within the entry (requires --entry)")
	c.cmd.Flags().StringVarP(&c.outDir, "output", "o", ".", "Output directory")
	return c
}

func (c *attachmentsDownloadCommand) run(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}
	threadID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return output.ErrUsage(fmt.Sprintf("invalid thread ID: %s", args[0]))
	}
	if c.index != 0 && c.entryID == 0 {
		return output.ErrUsage("--index requires --entry")
	}
	if c.index < 0 {
		return output.ErrUsage("--index must be >= 1")
	}

	entries, err := fetchThreadEntries(cmd.Context(), threadID)
	if err != nil {
		return err
	}

	type job struct {
		EntryID    int64
		Index      int
		Attachment models.Attachment
	}
	var jobs []job
	for _, e := range entries {
		if c.entryID != 0 && e.ID != c.entryID {
			continue
		}
		for i, att := range e.Attachments {
			if c.index != 0 && (i+1) != c.index {
				continue
			}
			jobs = append(jobs, job{EntryID: e.ID, Index: i + 1, Attachment: att})
		}
	}
	if len(jobs) == 0 {
		switch {
		case c.entryID != 0 && c.index != 0:
			return output.ErrNotFound("attachment", fmt.Sprintf("entry %d index %d", c.entryID, c.index))
		case c.entryID != 0:
			return output.ErrNotFound("attachments", fmt.Sprintf("entry %d", c.entryID))
		default:
			return output.ErrNotFound("attachments", fmt.Sprintf("thread %d", threadID))
		}
	}

	if err := os.MkdirAll(c.outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if info, statErr := os.Stat(c.outDir); statErr != nil || !info.IsDir() {
		return fmt.Errorf("output path is not a directory: %s", c.outDir)
	}

	type result struct {
		Path     string `json:"path"`
		Filename string `json:"filename"`
		EntryID  int64  `json:"entry_id"`
		Bytes    int    `json:"bytes"`
	}
	results := make([]result, 0, len(jobs))
	for _, j := range jobs {
		safe := sanitizeFilename(j.Attachment.Filename, j.Index)
		dest := uniquePath(filepath.Join(c.outDir, safe))
		resp, err := sdk.GetBlob(cmd.Context(), j.Attachment.URL)
		if err != nil {
			return convertSDKError(err)
		}
		if err := os.WriteFile(dest, resp.Data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
		results = append(results, result{
			Path:     dest,
			Filename: filepath.Base(dest),
			EntryID:  j.EntryID,
			Bytes:    len(resp.Data),
		})
		if writer.IsStyled() {
			fmt.Fprintf(cmd.OutOrStdout(), "%s (%d bytes)\n", dest, len(resp.Data))
		}
	}

	if writer.IsStyled() {
		return nil
	}
	return writeOK(results,
		output.WithSummary(fmt.Sprintf("downloaded %d attachment(s) to %s", len(results), c.outDir)),
	)
}

func fetchThreadEntries(ctx context.Context, threadID int64) ([]models.Entry, error) {
	resp, err := sdk.GetHTML(ctx, fmt.Sprintf("/topics/%d/entries", threadID))
	if err != nil {
		return nil, convertSDKError(err)
	}
	return htmlutil.ParseTopicEntriesHTML(string(resp.Data)), nil
}

func senderDisplay(e models.Entry) string {
	if e.AlternativeSenderName != "" {
		return e.AlternativeSenderName
	}
	if e.Creator.Name != "" {
		return e.Creator.Name
	}
	return e.Creator.EmailAddress
}

func shortDate(s string) string {
	if len(s) >= 16 {
		return s[:16]
	}
	return s
}

// sanitizeFilename returns a filename safe to join with an output directory.
// Reject anything containing path separators or matching . / .. — HTML is
// untrusted input. Fall back to "attachment-<index>" for empty or unsafe names.
func sanitizeFilename(name string, fallbackIndex int) string {
	clean := strings.TrimSpace(name)
	if clean == "" || clean == "." || clean == ".." {
		return fmt.Sprintf("attachment-%d", fallbackIndex)
	}
	if strings.ContainsAny(clean, `/\`) {
		return fmt.Sprintf("attachment-%d", fallbackIndex)
	}
	if clean != filepath.Base(clean) {
		return fmt.Sprintf("attachment-%d", fallbackIndex)
	}
	return clean
}

// uniquePath returns p, or p with -1, -2, ... inserted before the extension
// to avoid clobbering an existing file. If Stat returns any error (typically
// NotExist) we treat the path as available; a real I/O error will surface
// when WriteFile runs.
func uniquePath(p string) string {
	if _, err := os.Stat(p); err != nil {
		return p
	}
	dir, base := filepath.Split(p)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for i := 1; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s-%d%s", stem, i, ext))
		if _, err := os.Stat(candidate); err != nil {
			return candidate
		}
	}
}
