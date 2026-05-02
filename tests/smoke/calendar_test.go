package smoke_test

import (
	"encoding/json"
	"testing"
)

func TestCalendarsList(t *testing.T) {
	resp := heyJSON(t, "calendars")

	type Calendar struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Kind  string `json:"kind"`
		Owned bool   `json:"owned"`
	}
	calendars := dataAs[[]Calendar](t, resp)

	if len(calendars) == 0 {
		t.Fatal("expected at least one calendar")
	}

	// Should have a personal calendar.
	found := false
	for _, c := range calendars {
		if c.Owned {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one owned calendar")
	}

	// Cross-verify: the calendar page should be accessible and non-empty.
	html := fetchHTML(t, baseURL+"/calendar")
	if len(html) == 0 {
		t.Error("calendar page returned empty HTML")
	}
}

func TestRecordings(t *testing.T) {
	// Get a calendar ID first.
	resp := heyJSON(t, "calendars")
	type Calendar struct {
		ID int `json:"id"`
	}
	calendars := dataAs[[]Calendar](t, resp)
	if len(calendars) == 0 {
		t.Fatal("no calendars available")
	}

	calID := calendars[0].ID
	recResp := heyJSON(t, "recordings", intStr(calID))
	// Recordings response is a map of type → []recording.
	// Just verify the command succeeds and returns a map.
	if recResp.Data == nil || string(recResp.Data) == "null" {
		// Empty recordings returns null — that's acceptable.
		return
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(recResp.Data, &data); err != nil {
		t.Fatalf("recordings data is not a map: %v", err)
	}

	// Cross-verify: if there are recordings, pick one and verify its title
	// exists on the calendar page.
	type Recording struct {
		Title string `json:"title"`
	}
	for _, raw := range data {
		var recordings []Recording
		if err := json.Unmarshal(raw, &recordings); err != nil {
			continue
		}
		if len(recordings) > 0 && recordings[0].Title != "" {
			html := fetchHTML(t, baseURL+"/calendar")
			assertContains(t, html, recordings[0].Title)
			break
		}
	}
}

func TestRecordingsWithDateRange(t *testing.T) {
	resp := heyJSON(t, "calendars")
	type Calendar struct {
		ID int `json:"id"`
	}
	calendars := dataAs[[]Calendar](t, resp)
	if len(calendars) == 0 {
		t.Fatal("no calendars available")
	}

	calID := calendars[0].ID
	heyJSON(t, "recordings", intStr(calID),
		"--starts-on", "2024-01-01",
		"--ends-on", "2024-12-31",
	)
}

func TestRecordingsLimit(t *testing.T) {
	resp := heyJSON(t, "calendars")
	type Calendar struct {
		ID int `json:"id"`
	}
	calendars := dataAs[[]Calendar](t, resp)
	if len(calendars) == 0 {
		t.Fatal("no calendars available")
	}

	calID := calendars[0].ID
	heyJSON(t, "recordings", intStr(calID), "--limit", "5")
}

func TestRecordingsAll(t *testing.T) {
	resp := heyJSON(t, "calendars")
	type Calendar struct {
		ID int `json:"id"`
	}
	calendars := dataAs[[]Calendar](t, resp)
	if len(calendars) == 0 {
		t.Fatal("no calendars available")
	}

	calID := calendars[0].ID
	heyJSON(t, "recordings", intStr(calID), "--all")
}

func TestRecordingsNoArgument(t *testing.T) {
	heyFail(t, "recordings", "--json")
}

func TestRecordingsInvalidCalendarID(t *testing.T) {
	heyFail(t, "recordings", "999999999", "--json")
}

// TestEventCreateWithInvitees walks the full create→read→update→delete cycle
// for events with invitees, asserting that:
//   - --invitee on create attaches the email as an attendance.
//   - hey recordings exposes the attendance with email_address populated.
//   - --invitee on update replaces the full list (the dropped emails go away).
// Cleanup happens via t.Cleanup so the test calendar isn't littered with
// abandoned events even when intermediate steps fail. Skips gracefully when
// no owned calendar exists or when the dev server doesn't accept the create.
func TestEventCreateWithInvitees(t *testing.T) {
	resp := heyJSON(t, "calendars")
	type Calendar struct {
		ID    int  `json:"id"`
		Owned bool `json:"owned"`
	}
	calendars := dataAs[[]Calendar](t, resp)
	var calID int
	for _, c := range calendars {
		if c.Owned {
			calID = c.ID
			break
		}
	}
	if calID == 0 {
		t.Skip("no owned calendar available")
	}

	uid := uniqueID()
	title := "smoke-invitees-" + uid
	first := uid + "-a@example.com"
	second := uid + "-b@example.com"
	replacement := uid + "-c@example.com"

	createOut, createErr, createCode := hey(t, "event", "create", title,
		"--calendar-id", intStr(calID),
		"--starts-at", "2099-12-31",
		"--all-day",
		"--invitee", first,
		"--invitee", second,
		"--json",
	)
	if createCode != 0 {
		t.Skipf("create failed (exit %d): %s", createCode, createErr)
	}

	var createResp Response
	if err := json.Unmarshal([]byte(createOut), &createResp); err != nil {
		t.Fatalf("parse create response: %v", err)
	}
	if !createResp.OK {
		t.Fatal("create returned ok=false")
	}
	type IDPayload struct {
		ID int `json:"id"`
	}
	idData := dataAs[IDPayload](t, createResp)
	eventID := idData.ID
	if eventID == 0 {
		t.Fatal("no event ID returned from create")
	}
	t.Cleanup(func() {
		hey(t, "event", "delete", intStr(eventID))
	})

	// Read back via recordings — the event should exist with both invitees.
	recordings := heyJSON(t, "recordings", intStr(calID),
		"--starts-on", "2099-12-30",
		"--ends-on", "2100-01-02",
	)
	type Attendance struct {
		EmailAddress string `json:"email_address"`
		Status       string `json:"status"`
	}
	type Event struct {
		ID          int          `json:"id"`
		Title       string       `json:"title"`
		Attendances []Attendance `json:"attendances"`
	}
	var grouped map[string]json.RawMessage
	if err := json.Unmarshal(recordings.Data, &grouped); err != nil {
		t.Fatalf("recordings data is not a map: %v", err)
	}
	var events []Event
	if raw, ok := grouped["Calendar::Event"]; ok {
		if err := json.Unmarshal(raw, &events); err != nil {
			t.Fatalf("parse Calendar::Event: %v", err)
		}
	}
	var got *Event
	for i := range events {
		if events[i].ID == eventID {
			got = &events[i]
			break
		}
	}
	if got == nil {
		t.Fatalf("event %d not found in recordings", eventID)
	}

	emails := map[string]bool{}
	for _, a := range got.Attendances {
		emails[a.EmailAddress] = true
	}
	if !emails[first] {
		t.Errorf("attendances missing %q, got %+v", first, got.Attendances)
	}
	if !emails[second] {
		t.Errorf("attendances missing %q, got %+v", second, got.Attendances)
	}

	// Replace the invitee list with a single new email.
	_, updErr, updCode := hey(t, "event", "update", intStr(eventID),
		"--invitee", replacement,
		"--json",
	)
	if updCode != 0 {
		t.Fatalf("update failed (exit %d): %s", updCode, updErr)
	}

	// Re-read; the previous two should be gone, only `replacement` should remain.
	recordings2 := heyJSON(t, "recordings", intStr(calID),
		"--starts-on", "2099-12-30",
		"--ends-on", "2100-01-02",
	)
	if err := json.Unmarshal(recordings2.Data, &grouped); err != nil {
		t.Fatalf("recordings data is not a map: %v", err)
	}
	events = nil
	if raw, ok := grouped["Calendar::Event"]; ok {
		if err := json.Unmarshal(raw, &events); err != nil {
			t.Fatalf("parse Calendar::Event after update: %v", err)
		}
	}
	got = nil
	for i := range events {
		if events[i].ID == eventID {
			got = &events[i]
			break
		}
	}
	if got == nil {
		t.Fatalf("event %d not found after update", eventID)
	}
	emails = map[string]bool{}
	for _, a := range got.Attendances {
		emails[a.EmailAddress] = true
	}
	if !emails[replacement] {
		t.Errorf("attendances missing replacement %q after update, got %+v", replacement, got.Attendances)
	}
	if emails[first] || emails[second] {
		t.Errorf("update did not replace invitees — old emails still present: %+v", got.Attendances)
	}
}
