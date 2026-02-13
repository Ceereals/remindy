package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/abenz1267/elephant/v2/pkg/common"
	"github.com/abenz1267/elephant/v2/pkg/pb/pb"
)

var (
	Name       = "remindy"
	NamePretty = "Reminders"
	config     *Config
	reminders  []Reminder
	creating   bool
	mu         sync.Mutex
)

type Config struct {
	common.Config `koanf:",squash"`
	Location      string `koanf:"location" desc:"path to reminders.json" default:"~/.local/share/remindy/reminders.json"`
}

type Reminder struct {
	ID           string `json:"id"`
	Text         string `json:"text"`
	Time         string `json:"time"`
	Type         string `json:"type"`
	Days         []int  `json:"days,omitempty"`
	Created      string `json:"created"`
	Notified     bool   `json:"notified,omitempty"`
	LastNotified string `json:"last_notified,omitempty"`
}

type RemindersFile struct {
	Reminders []Reminder `json:"reminders"`
}

var dayNames = map[int]string{
	1: "Mon", 2: "Tue", 3: "Wed", 4: "Thu",
	5: "Fri", 6: "Sat", 7: "Sun",
}

var dayNumbers = map[string]int{
	"monday": 1, "tuesday": 2, "wednesday": 3, "thursday": 4,
	"friday": 5, "saturday": 6, "sunday": 7,
	"mon": 1, "tue": 2, "wed": 3, "thu": 4,
	"fri": 5, "sat": 6, "sun": 7,
}

func remindersPath() string {
	if config != nil && config.Location != "" {
		loc := config.Location
		if strings.HasPrefix(loc, "~/") {
			home, _ := os.UserHomeDir()
			loc = filepath.Join(home, loc[2:])
		}
		return loc
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "remindy", "reminders.json")
}

func lockPath() string {
	return fmt.Sprintf("/tmp/remindy-%d.lock", os.Getuid())
}

func withLock(fn func() error) error {
	lf, err := os.OpenFile(lockPath(), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lf.Close()

	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lf.Fd()), syscall.LOCK_UN)

	return fn()
}

func loadReminders() {
	mu.Lock()
	defer mu.Unlock()

	reminders = nil

	_ = withLock(func() error {
		data, err := os.ReadFile(remindersPath())
		if err != nil {
			return err
		}

		var rf RemindersFile
		if err := json.Unmarshal(data, &rf); err != nil {
			return err
		}

		reminders = rf.Reminders
		return nil
	})
}

func saveReminders() {
	_ = withLock(func() error {
		rf := RemindersFile{Reminders: reminders}
		data, err := json.MarshalIndent(rf, "", "  ")
		if err != nil {
			return err
		}

		path := remindersPath()
		tmp := fmt.Sprintf("%s.tmp.%d", path, os.Getpid())

		if err := os.WriteFile(tmp, data, 0o644); err != nil {
			os.Remove(tmp)
			return err
		}

		if err := os.Rename(tmp, path); err != nil {
			os.Remove(tmp)
			return err
		}

		return nil
	})
}

func genID() string {
	b := make([]byte, 4)
	f, err := os.Open("/dev/urandom")
	if err != nil {
		return fmt.Sprintf("%08x", time.Now().UnixNano())
	}
	defer f.Close()
	f.Read(b)
	return fmt.Sprintf("%02x%02x%02x%02x", b[0], b[1], b[2], b[3])
}

// Time parsing

var durationRe = regexp.MustCompile(`^(?:(\d+)w)?(?:(\d+)d)?(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$`)

func parseDuration(s string) (time.Duration, bool) {
	s = strings.TrimSpace(s)
	m := durationRe.FindStringSubmatch(s)
	if m == nil || s == "" {
		return 0, false
	}

	var d time.Duration
	if m[1] != "" {
		n, _ := strconv.Atoi(m[1])
		d += time.Duration(n) * 7 * 24 * time.Hour
	}
	if m[2] != "" {
		n, _ := strconv.Atoi(m[2])
		d += time.Duration(n) * 24 * time.Hour
	}
	if m[3] != "" {
		n, _ := strconv.Atoi(m[3])
		d += time.Duration(n) * time.Hour
	}
	if m[4] != "" {
		n, _ := strconv.Atoi(m[4])
		d += time.Duration(n) * time.Minute
	}
	if m[5] != "" {
		n, _ := strconv.Atoi(m[5])
		d += time.Duration(n) * time.Second
	}

	return d, d > 0
}

// durationToBashFormat converts Go plugin duration (e.g. "1w2d3h30m5s")
// to remindy-add format (e.g. "9D3h30m5s") where weeks become days and d→D.
func durationToBashFormat(s string) string {
	m := durationRe.FindStringSubmatch(s)
	if m == nil {
		return s
	}

	var result string
	totalDays := 0
	if m[1] != "" {
		n, _ := strconv.Atoi(m[1])
		totalDays += n * 7
	}
	if m[2] != "" {
		n, _ := strconv.Atoi(m[2])
		totalDays += n
	}
	if totalDays > 0 {
		result += strconv.Itoa(totalDays) + "D"
	}
	if m[3] != "" {
		result += m[3] + "h"
	}
	if m[4] != "" {
		result += m[4] + "m"
	}
	if m[5] != "" {
		result += m[5] + "s"
	}
	return result
}

func parseHHMM(s string) (int, int, bool) {
	parts := strings.SplitN(strings.TrimSpace(s), ":", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

type parsedReminder struct {
	text    string
	rtype   string
	absTime time.Time // for "once" with "at"
	durStr  string    // for "once" with "in" (original duration string)
	hhmm    string    // for daily/weekly
	days    []int     // for weekly
	label   string    // display label
}

func parseCreateQuery(query string) *parsedReminder {
	// Split on ">" to get schedule and text
	parts := strings.SplitN(query, ">", 2)
	if len(parts) != 2 {
		return nil
	}

	schedule := strings.TrimSpace(parts[0])
	text := strings.TrimSpace(parts[1])
	if text == "" || schedule == "" {
		return nil
	}

	lower := strings.ToLower(schedule)

	// "every day at HH:MM" or "every <daylist> at HH:MM"
	if strings.HasPrefix(lower, "every ") {
		rest := strings.TrimPrefix(lower, "every ")

		atIdx := strings.LastIndex(rest, " at ")
		if atIdx < 0 {
			return nil
		}

		daysPart := strings.TrimSpace(rest[:atIdx])
		timePart := strings.TrimSpace(rest[atIdx+4:])

		h, m, ok := parseHHMM(timePart)
		if !ok {
			return nil
		}

		hhmm := fmt.Sprintf("%02d:%02d", h, m)

		if daysPart == "day" {
			return &parsedReminder{
				text:  text,
				rtype: "daily",
				hhmm:  hhmm,
				label: fmt.Sprintf("every day at %s", hhmm),
			}
		}

		// Parse comma-separated days
		dayStrs := strings.Split(daysPart, ",")
		var days []int
		var dayLabels []string
		for _, ds := range dayStrs {
			ds = strings.TrimSpace(ds)
			if n, ok := dayNumbers[ds]; ok {
				days = append(days, n)
				dayLabels = append(dayLabels, dayNames[n])
			}
		}

		if len(days) == 0 {
			return nil
		}

		return &parsedReminder{
			text:  text,
			rtype: "weekly",
			hhmm:  hhmm,
			days:  days,
			label: fmt.Sprintf("every %s at %s", strings.Join(dayLabels, ","), hhmm),
		}
	}

	// "in <duration>"
	if strings.HasPrefix(lower, "in ") {
		durStr := strings.TrimPrefix(lower, "in ")
		dur, ok := parseDuration(durStr)
		if !ok {
			return nil
		}

		t := time.Now().Add(dur)
		return &parsedReminder{
			text:    text,
			rtype:   "once",
			absTime: t,
			durStr:  durStr,
			label:   schedule,
		}
	}

	// "at HH:MM"
	if strings.HasPrefix(lower, "at ") {
		timeStr := strings.TrimPrefix(lower, "at ")
		h, m, ok := parseHHMM(timeStr)
		if !ok {
			return nil
		}

		now := time.Now()
		t := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, now.Location())
		if t.Before(now) {
			t = t.AddDate(0, 0, 1)
		}

		return &parsedReminder{
			text:    text,
			rtype:   "once",
			absTime: t,
			label:   schedule,
		}
	}

	return nil
}

func reminderSubtext(r Reminder) string {
	switch r.Type {
	case "once":
		t, err := time.Parse("2006-01-02T15:04:05", r.Time)
		if err != nil {
			return "once · " + r.Time
		}
		return fmt.Sprintf("once · %s at %s", t.Format("2006-01-02"), t.Format("15:04"))
	case "daily":
		return "daily · every day at " + r.Time
	case "weekly":
		var names []string
		for _, d := range r.Days {
			if n, ok := dayNames[d]; ok {
				names = append(names, n)
			}
		}
		return fmt.Sprintf("weekly · every %s at %s", strings.Join(names, ","), r.Time)
	}
	return r.Type
}

func Setup() {
	config = &Config{
		Config: common.Config{
			Icon:     "alarm-symbolic",
			MinScore: 20,
		},
		Location: "~/.local/share/remindy/reminders.json",
	}

	common.LoadConfig(Name, config)

	if config.NamePretty != "" {
		NamePretty = config.NamePretty
	}

	loadReminders()
}

func Available() bool {
	_, err := os.Stat(remindersPath())
	return err == nil
}

func Icon() string {
	return config.Icon
}

func HideFromProviderlist() bool {
	return config.HideFromProviderlist
}

func PrintDoc() {
	fmt.Println("# Remindy")
	fmt.Println()
	fmt.Println("Reminder provider for elephant. Reads/writes ~/.local/share/remindy/reminders.json")
	fmt.Println()
	fmt.Println("## Quick-add syntax")
	fmt.Println()
	fmt.Println("  in 30m > meeting")
	fmt.Println("  at 14:30 > standup")
	fmt.Println("  every day at 09:00 > coffee")
	fmt.Println("  every monday,friday at 10:00 > review")
}

func State(_ string) *pb.ProviderStateResponse {
	states := []string{}
	actions := []string{}

	if creating {
		states = append(states, "creating")
		actions = append(actions, "search")
	} else {
		states = append(states, "searching")
		actions = append(actions, "create")
	}

	return &pb.ProviderStateResponse{
		States:  states,
		Actions: actions,
	}
}

func Query(_ net.Conn, query string, _ bool, exact bool, _ uint8) []*pb.QueryResponse_Item {
	loadReminders()

	mu.Lock()
	items := make([]Reminder, len(reminders))
	copy(items, reminders)
	mu.Unlock()

	entries := []*pb.QueryResponse_Item{}

	// Check for fast-create via ">"
	parsed := parseCreateQuery(query)
	fastCreate := parsed != nil && !creating

	if creating || fastCreate {
		if parsed != nil {
			e := &pb.QueryResponse_Item{
				Provider:   Name,
				Identifier: fmt.Sprintf("CREATE:%s", query),
				Icon:       "list-add-symbolic",
				Text:       fmt.Sprintf("Add: %s %s", parsed.text, parsed.label),
				Subtext:    parsed.rtype,
				Score:      3_000_000,
				Actions:    []string{"save"},
				State:      []string{"creating"},
				Fuzzyinfo:  &pb.QueryResponse_Item_FuzzyInfo{},
			}
			entries = append(entries, e)
		} else if creating && strings.TrimSpace(query) != "" {
			// In create mode but can't parse yet - show hint
			e := &pb.QueryResponse_Item{
				Provider:   Name,
				Identifier: "",
				Icon:       "dialog-information-symbolic",
				Text:       "Type: in 30m > text / at 14:30 > text / every day at 09:00 > text",
				Subtext:    "schedule > reminder text",
				Score:      3_000_000,
				State:      []string{"creating"},
				Fuzzyinfo:  &pb.QueryResponse_Item_FuzzyInfo{},
			}
			entries = append(entries, e)
		}

		if fastCreate {
			// Also show search results below
			searchQuery := ""
			searchEntries := searchReminders(items, searchQuery, exact)
			entries = append(entries, searchEntries...)
		}

		return entries
	}

	// Search mode
	entries = searchReminders(items, query, exact)

	// Always show "Add reminder..." at the bottom
	if query == "" {
		entries = append(entries, &pb.QueryResponse_Item{
			Provider:   Name,
			Identifier: "LAUNCH_ADD",
			Icon:       "list-add-symbolic",
			Text:       "Add reminder...",
			Subtext:    "open interactive reminder creator",
			Score:      1,
			Actions:    []string{"activate"},
			Fuzzyinfo:  &pb.QueryResponse_Item_FuzzyInfo{},
		})
	}

	return entries
}

func searchReminders(items []Reminder, query string, exact bool) []*pb.QueryResponse_Item {
	now := time.Now()
	entries := []*pb.QueryResponse_Item{}

	// Sort: upcoming once first, then daily, then weekly
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]

		typeOrder := map[string]int{"once": 0, "daily": 1, "weekly": 2}
		ao, bo := typeOrder[a.Type], typeOrder[b.Type]

		if a.Type == "once" && b.Type == "once" {
			ta, _ := time.Parse("2006-01-02T15:04:05", a.Time)
			tb, _ := time.Parse("2006-01-02T15:04:05", b.Time)
			if !ta.IsZero() && !tb.IsZero() {
				return ta.Before(tb)
			}
		}

		return ao < bo
	})

	for i, r := range items {
		// Skip notified once reminders
		if r.Type == "once" && r.Notified {
			continue
		}

		e := &pb.QueryResponse_Item{
			Provider:   Name,
			Identifier: strconv.Itoa(i),
			Icon:       "alarm-symbolic",
			Text:       r.Text,
			Subtext:    reminderSubtext(r),
			Score:      int32(999_999 - i),
			Actions:    []string{"delete"},
			Fuzzyinfo:  &pb.QueryResponse_Item_FuzzyInfo{},
		}

		// Add urgency state for upcoming once reminders
		if r.Type == "once" {
			if t, err := time.Parse("2006-01-02T15:04:05", r.Time); err == nil {
				diff := time.Until(t)
				if diff > 0 && diff < 10*time.Minute {
					e.State = []string{"urgent"}
					e.Score = 2_000_000 + int32(diff.Seconds())
				} else if diff > 0 && diff < time.Hour {
					e.Subtext += fmt.Sprintf(" (in %dm)", int(diff.Minutes()))
				} else if t.Format("2006-01-02") == now.Format("2006-01-02") {
					e.Subtext += " (today)"
				}
			}
		}

		if query != "" {
			e.Score, e.Fuzzyinfo.Positions, e.Fuzzyinfo.Start = common.FuzzyScore(query, e.Text, exact)
			if e.Score <= config.MinScore {
				continue
			}
		}

		entries = append(entries, e)
	}

	return entries
}

func Activate(_ bool, identifier, action, query, args string, _ uint8, _ net.Conn) {
	switch action {
	case "activate":
		if identifier == "LAUNCH_ADD" {
			cmd := exec.Command("omarchy-launch-tui", "remindy-add")
			if err := cmd.Start(); err != nil {
				slog.Error(Name, "activate", "failed to launch remindy-add", "error", err)
			}
		}
	case "search":
		creating = false
	case "create":
		creating = true
	case "save":
		creating = false
		after, ok := strings.CutPrefix(identifier, "CREATE:")
		if !ok {
			return
		}
		createReminder(after)
	case "delete":
		idx, err := strconv.Atoi(identifier)
		if err != nil {
			slog.Error(Name, "delete", err)
			return
		}
		deleteReminder(idx)
	default:
		slog.Error(Name, "activate", fmt.Sprintf("unknown action: %s", action))
	}
}

func createReminder(query string) {
	parsed := parseCreateQuery(query)
	if parsed == nil {
		slog.Error(Name, "create", "failed to parse query", "query", query)
		return
	}

	var args []string
	args = append(args, parsed.text)

	switch parsed.rtype {
	case "once":
		if parsed.durStr != "" {
			args = append(args, "in", durationToBashFormat(parsed.durStr))
		} else {
			args = append(args, "at", parsed.absTime.Format("2006-01-02 15:04"))
		}
	case "daily":
		args = append(args, "every", "day", "at", parsed.hhmm)
	case "weekly":
		var dayLabels []string
		for _, d := range parsed.days {
			if n, ok := dayNames[d]; ok {
				dayLabels = append(dayLabels, strings.ToLower(n))
			}
		}
		args = append(args, "every", strings.Join(dayLabels, ","), "at", parsed.hhmm)
	}

	cmd := exec.Command("remindy-add", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		slog.Error(Name, "create", "remindy-add failed", "error", err, "output", string(out))
		return
	}

	loadReminders()
}

func deleteReminder(idx int) {
	mu.Lock()
	defer mu.Unlock()

	if idx < 0 || idx >= len(reminders) {
		slog.Error(Name, "delete", "index out of range", "idx", idx)
		return
	}

	reminders = append(reminders[:idx], reminders[idx+1:]...)
	saveReminders()
}
