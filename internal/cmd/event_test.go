package cmd

import (
	"reflect"
	"testing"
)

func TestEventCreateInviteeFlagRepeated(t *testing.T) {
	c := newEventCreateCommand()
	if err := c.cmd.ParseFlags([]string{
		"--invitee", "alice@example.com",
		"--invitee", "bob@example.com",
	}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	want := []string{"alice@example.com", "bob@example.com"}
	if !reflect.DeepEqual(c.invitees, want) {
		t.Errorf("invitees = %v, want %v", c.invitees, want)
	}
}

func TestEventCreateInviteeFlagCommaSeparated(t *testing.T) {
	c := newEventCreateCommand()
	if err := c.cmd.ParseFlags([]string{
		"--invitee", "alice@example.com,bob@example.com",
	}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	want := []string{"alice@example.com", "bob@example.com"}
	if !reflect.DeepEqual(c.invitees, want) {
		t.Errorf("invitees = %v, want %v", c.invitees, want)
	}
}

func TestEventCreateInviteeFlagDefault(t *testing.T) {
	c := newEventCreateCommand()
	if err := c.cmd.ParseFlags([]string{}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if c.invitees != nil {
		t.Errorf("invitees default = %v, want nil", c.invitees)
	}
	if c.cmd.Flags().Changed("invitee") {
		t.Error("Changed(invitee) = true on default, want false")
	}
}

func TestEventUpdateInviteeFlagSetVsUnset(t *testing.T) {
	// Unset: Changed("invitee") must be false (params.Invitees stays nil → SDK skips).
	cUnset := newEventUpdateCommand()
	if err := cUnset.cmd.ParseFlags([]string{"--title", "renamed"}); err != nil {
		t.Fatalf("ParseFlags unset: %v", err)
	}
	if cUnset.cmd.Flags().Changed("invitee") {
		t.Error("Changed(invitee) = true when only --title was passed")
	}

	// Set: Changed("invitee") must be true with the parsed slice.
	cSet := newEventUpdateCommand()
	if err := cSet.cmd.ParseFlags([]string{
		"--invitee", "carol@example.com",
		"--invitee", "dan@example.com",
	}); err != nil {
		t.Fatalf("ParseFlags set: %v", err)
	}
	if !cSet.cmd.Flags().Changed("invitee") {
		t.Error("Changed(invitee) = false after passing the flag")
	}
	want := []string{"carol@example.com", "dan@example.com"}
	if !reflect.DeepEqual(cSet.invitees, want) {
		t.Errorf("invitees = %v, want %v", cSet.invitees, want)
	}
}
