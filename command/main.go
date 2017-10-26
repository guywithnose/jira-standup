package command

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/urfave/cli"
)

// CmdMain summarizes the tickets the user has recently tracked time against
func CmdMain(c *cli.Context) error {
	if c.NArg() != 1 && c.NArg() != 0 {
		return cli.NewExitError("Usage \"jira-standup {date}\"", 1)
	}

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

	date, err := handleDate(c.Args().Get(0))
	if err != nil {
		return err
	}

	client, err := getClient(url, username, password)
	if err != nil {
		return fmt.Errorf("Unable to get client: %v", err)
	}

	return printDurations(client, username, date, c.App.Writer)
}

func printDurations(client *jira.Client, username string, date time.Time, writer io.Writer) error {
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

	tabW := tabwriter.NewWriter(writer, 0, 0, 1, ' ', 0)
	for _, id := range sortedIssues {
		fmt.Fprintf(tabW, "%v\t%s\n", durations[id], id)
		total += durations[id]
	}

	_ = tabW.Flush()

	fmt.Fprintf(writer, "Total: %v\n", total)
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

func handleDate(dateParam string) (time.Time, error) {
	relativeDate, err := strconv.Atoi(dateParam)
	if err == nil {
		dateParam = time.Now().AddDate(0, 0, -relativeDate).Format("2006-01-02")
	}

	if dateParam == "" {
		dateParam = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	}

	return time.Parse("2006-01-02", dateParam)
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

	return parseDurations(client, issues, username, date)
}

func parseDurations(client *jira.Client, issues []jira.Issue, username string, date time.Time) (map[string]time.Duration, error) {
	durations := map[string]time.Duration{}
	for _, issue := range issues {
		issueKey := fmt.Sprintf("%s\t%s", issue.Key, issue.Fields.Summary)
		durations[issueKey] = time.Duration(0)
		if issue.Fields != nil && issue.Fields.Worklog != nil {
			var worklogs []jira.WorklogRecord
			if issue.Fields.Worklog.MaxResults != issue.Fields.Worklog.Total {
				worklog, _, err := client.Issue.GetWorklogs(issue.Key)
				if err != nil {
					return nil, fmt.Errorf("Unable to make worklog call: %v", err)
				}

				worklogs = worklog.Worklogs
			} else {
				worklogs = issue.Fields.Worklog.Worklogs
			}

			for _, wl := range worklogs {
				if wl.Author.Name == username && time.Time(wl.Created).After(date) && time.Time(wl.Created).Before(date.Add(24*time.Hour)) {
					duration, _ := time.ParseDuration(fmt.Sprintf("%ds", wl.TimeSpentSeconds))
					durations[issueKey] += duration
				}
			}
		}
	}

	return durations, nil
}
