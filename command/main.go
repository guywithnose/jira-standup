package command

import (
	"fmt"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/urfave/cli"
)

// CmdMain does something
func CmdMain(c *cli.Context) error {
	username := c.String("username")
	if username == "" {
		return cli.NewExitError("You must specify --username", 1)
	}

	password := c.String("password")
	if password == "" {
		return cli.NewExitError("You must specify --password", 1)
	}

	url := c.String("url")
	if url == "" {
		return cli.NewExitError("You must specify --url", 1)
	}

	dateString := c.String("date")
	relativeDate := c.Int("relativeDate")

	date, err := handleDate(dateString, relativeDate)
	if err != nil {
		return err
	}

	client, err := getClient(url, username, password)
	if err != nil {
		return fmt.Errorf("Unable to get client: %v", err)
	}

	durations, err := getDurations(client, username, date)
	if err != nil {
		return fmt.Errorf("Unable to get durations: %v", err)
	}

	total := time.Duration(0)
	sortedIssues := []string{}
	for id := range durations {
		sortedIssues = append(sortedIssues, id)
	}

	sort.Strings(sortedIssues)

	tabW := tabwriter.NewWriter(c.App.Writer, 0, 0, 1, ' ', tabwriter.Debug)
	for _, id := range sortedIssues {
		fmt.Fprintf(tabW, "%v\t%s\n", durations[id], id)
		total += durations[id]
	}

	_ = tabW.Flush()

	fmt.Fprintf(c.App.Writer, "Total: %v\n", total)
	return nil
}

func getClient(url, username, password string) (*jira.Client, error) {
	client, err := jira.NewClient(nil, url)
	if err != nil {
		return nil, err
	}

	_, err = client.Authentication.AcquireSessionCookie(username, password)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func handleDate(dateString string, relativeDate int) (time.Time, error) {
	if relativeDate != 0 {
		if dateString != "" {
			return time.Time{}, cli.NewExitError("Error: Cannot specify date and relative date", 1)
		}

		dateString = time.Now().AddDate(0, 0, -relativeDate).Format("2006-01-02")
	}

	if dateString == "" {
		dateString = time.Now().Format("2006-01-02")
	}

	return time.Parse("2006-01-02", dateString)
}

func getDurations(client *jira.Client, username string, date time.Time) (map[string]time.Duration, error) {
	so := &jira.SearchOptions{
		StartAt:    0,
		MaxResults: 50,
		Fields:     []string{"*all"},
	}

	query := fmt.Sprintf("worklogDate = '%s' and worklogAuthor = %s", date.Format("2006-01-02"), username)

	issues, _, err := client.Issue.Search(query, so)
	if err != nil {
		return nil, fmt.Errorf("Unable to make search call: %v", err)
	}

	return parseDurations(issues, username, date)
}

func parseDurations(issues []jira.Issue, username string, date time.Time) (map[string]time.Duration, error) {
	durations := map[string]time.Duration{}
	for _, issue := range issues {
		issueKey := fmt.Sprintf("%s\t%s", issue.Key, issue.Fields.Summary)
		durations[issueKey] = time.Duration(0)
		if issue.Fields != nil && issue.Fields.Worklog != nil && issue.Fields.Worklog.Worklogs != nil {
			for _, wl := range issue.Fields.Worklog.Worklogs {
				if wl.Author.Name == username && time.Time(wl.Created).After(date) && time.Time(wl.Created).Before(date.Add(24*time.Hour)) {
					duration, _ := time.ParseDuration(fmt.Sprintf("%ds", wl.TimeSpentSeconds))
					durations[issueKey] += duration
				}
			}
		}
	}

	return durations, nil
}