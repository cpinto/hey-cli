package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	hey "github.com/basecamp/hey-sdk/go/pkg/hey"

	"github.com/basecamp/hey-cli/internal/output"
)

type eventCommand struct {
	cmd *cobra.Command
}

func newEventCommand() *eventCommand {
	eventCommand := &eventCommand{}
	eventCommand.cmd = &cobra.Command{
		Use:   "event",
		Short: "Manage calendar events",
		Annotations: map[string]string{
			"agent_notes": "Subcommands: create, update, delete. Use calendar ID from hey calendars and event IDs from hey recordings.",
		},
	}

	eventCommand.cmd.AddCommand(newEventCreateCommand().cmd)
	eventCommand.cmd.AddCommand(newEventUpdateCommand().cmd)
	eventCommand.cmd.AddCommand(newEventDeleteCommand().cmd)

	return eventCommand
}

// create

type eventCreateCommand struct {
	cmd        *cobra.Command
	title      string
	calendarID int64
	startsAt   string
	endsAt     string
	allDay     bool
	startTime  string
	endTime    string
	timeZone   string
	reminders  []string
}

func newEventCreateCommand() *eventCreateCommand {
	c := &eventCreateCommand{}
	c.cmd = &cobra.Command{
		Use:   "create [title]",
		Short: "Create a calendar event",
		Example: `  hey event create "Team standup" --calendar-id 1 --starts-at 2024-01-20 --start-time 09:00 --end-time 09:30 --timezone America/New_York
  hey event create -t "Day off" --calendar-id 1 --starts-at 2024-01-20 --all-day
  hey event create "Lunch" --calendar-id 1 --starts-at 2024-01-20 --start-time 12:00 --end-time 13:00 --timezone America/New_York --reminder 15m --reminder 5m`,
		RunE: c.run,
		Args: cobra.MaximumNArgs(1),
	}

	c.cmd.Flags().StringVarP(&c.title, "title", "t", "", "Event title")
	c.cmd.Flags().Int64Var(&c.calendarID, "calendar-id", 0, "Calendar ID (required)")
	c.cmd.Flags().StringVar(&c.startsAt, "starts-at", "", "Start date (YYYY-MM-DD, required)")
	c.cmd.Flags().StringVar(&c.endsAt, "ends-at", "", "End date (YYYY-MM-DD, defaults to starts-at)")
	c.cmd.Flags().BoolVar(&c.allDay, "all-day", false, "All-day event")
	c.cmd.Flags().StringVar(&c.startTime, "start-time", "", "Start time (HH:MM, required for timed events)")
	c.cmd.Flags().StringVar(&c.endTime, "end-time", "", "End time (HH:MM, required for timed events)")
	c.cmd.Flags().StringVar(&c.timeZone, "timezone", "", "IANA timezone (e.g. America/New_York, required for timed events)")
	c.cmd.Flags().StringSliceVar(&c.reminders, "reminder", nil, "Reminder duration before event (e.g. 15m, 1h), can be repeated")

	return c
}

func (c *eventCreateCommand) run(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}

	title := c.title
	if title != "" && len(args) > 0 {
		return output.ErrUsage("--title and positional argument are mutually exclusive")
	}
	if title == "" && len(args) > 0 {
		title = args[0]
	}
	if title == "" && !stdinIsTerminal() {
		var err error
		title, err = readStdin()
		if err != nil {
			return err
		}
	}
	if title == "" {
		return output.ErrUsageHint("title is required",
			`hey event create "Meeting" --calendar-id 1 --starts-at 2024-01-20 --start-time 09:00 --end-time 10:00 --timezone America/New_York`)
	}

	if c.calendarID == 0 {
		return output.ErrUsageHint("--calendar-id is required", "hey calendars --ids-only")
	}
	if c.startsAt == "" {
		return output.ErrUsageHint("--starts-at is required", "--starts-at 2024-01-20")
	}
	if !c.allDay && (c.startTime == "" || c.endTime == "") {
		return output.ErrUsageHint("--start-time and --end-time are required for timed events",
			"Use --all-day for all-day events, or provide --start-time and --end-time")
	}
	if !c.allDay && c.timeZone == "" {
		return output.ErrUsageHint("--timezone is required for timed events",
			"--timezone America/New_York")
	}

	reminders, err := parseReminders(c.reminders)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	id, err := sdk.CalendarEvents().Create(ctx, hey.CreateCalendarEventParams{
		CalendarID: c.calendarID,
		Title:      title,
		StartsAt:   c.startsAt,
		EndsAt:     c.endsAt,
		AllDay:     c.allDay,
		StartTime:  c.startTime,
		EndTime:    c.endTime,
		TimeZone:   c.timeZone,
		Reminders:  reminders,
	})
	if err != nil {
		return convertSDKError(err)
	}

	if writer.IsStyled() {
		fmt.Fprintf(cmd.OutOrStdout(), "Event created (ID: %d).\n", id)
		return nil
	}

	return writeOK(map[string]any{"id": id}, output.WithSummary("Event created"))
}

// update

type eventUpdateCommand struct {
	cmd       *cobra.Command
	title     string
	startsAt  string
	endsAt    string
	allDay    *bool
	startTime string
	endTime   string
	timeZone  string
	reminders []string
}

func newEventUpdateCommand() *eventUpdateCommand {
	c := &eventUpdateCommand{}
	c.cmd = &cobra.Command{
		Use:   "update <id>",
		Short: "Update a calendar event",
		Example: `  hey event update 123 --title "New title"
  hey event update 123 --starts-at 2024-01-21 --start-time 10:00 --end-time 11:00
  hey event update 123 --all-day`,
		RunE: c.run,
		Args: usageExactOneArg(),
	}

	c.cmd.Flags().StringVarP(&c.title, "title", "t", "", "Event title")
	c.cmd.Flags().StringVar(&c.startsAt, "starts-at", "", "Start date (YYYY-MM-DD)")
	c.cmd.Flags().StringVar(&c.endsAt, "ends-at", "", "End date (YYYY-MM-DD)")
	c.cmd.Flags().StringVar(&c.startTime, "start-time", "", "Start time (HH:MM)")
	c.cmd.Flags().StringVar(&c.endTime, "end-time", "", "End time (HH:MM)")
	c.cmd.Flags().StringVar(&c.timeZone, "timezone", "", "IANA timezone (e.g. America/New_York)")
	c.cmd.Flags().StringSliceVar(&c.reminders, "reminder", nil, "Reminder duration before event (e.g. 15m, 1h), can be repeated")

	// --all-day is a tristate: unset means don't change, --all-day means true, --no-all-day means false.
	// We handle this by checking cmd.Flags().Changed("all-day").
	c.cmd.Flags().Bool("all-day", false, "Make event all-day (use --all-day=false to make timed)")

	return c
}

func (c *eventUpdateCommand) run(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return output.ErrUsage(fmt.Sprintf("invalid event ID: %s", args[0]))
	}

	params := hey.UpdateCalendarEventParams{}

	if cmd.Flags().Changed("title") {
		params.Title = &c.title
	}
	if cmd.Flags().Changed("starts-at") {
		params.StartsAt = &c.startsAt
	}
	if cmd.Flags().Changed("ends-at") {
		params.EndsAt = &c.endsAt
	}
	if cmd.Flags().Changed("all-day") {
		v, _ := cmd.Flags().GetBool("all-day")
		params.AllDay = &v
	}
	if cmd.Flags().Changed("start-time") {
		params.StartTime = &c.startTime
	}
	if cmd.Flags().Changed("end-time") {
		params.EndTime = &c.endTime
	}
	if cmd.Flags().Changed("timezone") {
		params.TimeZone = &c.timeZone
	}
	if cmd.Flags().Changed("reminder") {
		reminders, rerr := parseReminders(c.reminders)
		if rerr != nil {
			return rerr
		}
		params.Reminders = reminders
	}

	ctx := cmd.Context()
	err = sdk.CalendarEvents().Update(ctx, id, params)
	if err != nil {
		return convertSDKError(err)
	}

	if writer.IsStyled() {
		fmt.Fprintln(cmd.OutOrStdout(), "Event updated.")
		return nil
	}

	return writeOK(nil, output.WithSummary("Event updated"))
}

// delete

type eventDeleteCommand struct {
	cmd *cobra.Command
}

func newEventDeleteCommand() *eventDeleteCommand {
	c := &eventDeleteCommand{}
	c.cmd = &cobra.Command{
		Use:     "delete <id>",
		Short:   "Delete a calendar event",
		Example: `  hey event delete 123`,
		RunE:    c.run,
		Args:    usageExactOneArg(),
	}

	return c
}

func (c *eventDeleteCommand) run(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return output.ErrUsage(fmt.Sprintf("invalid event ID: %s", args[0]))
	}

	ctx := cmd.Context()
	err = sdk.CalendarEvents().Delete(ctx, id)
	if err != nil {
		return convertSDKError(err)
	}

	if writer.IsStyled() {
		fmt.Fprintln(cmd.OutOrStdout(), "Event deleted.")
		return nil
	}

	return writeOK(nil, output.WithSummary("Event deleted"))
}

// parseReminders converts string durations (e.g. "15m", "1h") to time.Duration slices.
func parseReminders(raw []string) ([]time.Duration, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	reminders := make([]time.Duration, 0, len(raw))
	for _, s := range raw {
		d, err := time.ParseDuration(s)
		if err != nil {
			return nil, output.ErrUsage(fmt.Sprintf("invalid reminder duration %q: %v", s, err))
		}
		reminders = append(reminders, d)
	}
	return reminders, nil
}
