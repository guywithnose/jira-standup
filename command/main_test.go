package command_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/guywithnose/jira-standup/command"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdMain(t *testing.T) {
	ts := getMockJiraAPI(t, time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
	defer ts.Close()
	app, writer, set := getBaseAppAndFlagSet(ts.URL)
	assert.Nil(t, command.CmdMain(cli.NewContext(app, set, nil)))
	assert.Equal(
		t,
		"3h0m0s PROJ-12 Issue 12\n1h0m0s PROJ-13 Issue 13\nTotal: 4h0m0s\n",
		writer.String(),
	)
}

func TestCmdMainNoUsername(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.String("url", "url", "doc")
	set.String("password", "pw", "doc")
	app := cli.NewApp()
	assert.EqualError(t, command.CmdMain(cli.NewContext(app, set, nil)), "You must specify --username")
}

func TestCmdMainNoPassword(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.String("url", "url", "doc")
	set.String("username", "un", "doc")
	app := cli.NewApp()
	assert.EqualError(t, command.CmdMain(cli.NewContext(app, set, nil)), "You must specify --password")
}

func TestCmdMainNoUrl(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.String("username", "un", "doc")
	set.String("password", "pw", "doc")
	app := cli.NewApp()
	assert.EqualError(t, command.CmdMain(cli.NewContext(app, set, nil)), "You must specify --url")
}

func TestCmdMainDateOverride(t *testing.T) {
	ts := getMockJiraAPI(t, "2016-03-25")
	defer ts.Close()
	app, writer, set := getBaseAppAndFlagSet(ts.URL)
	assert.Nil(t, set.Parse([]string{"2016-03-25"}))
	assert.Nil(t, command.CmdMain(cli.NewContext(app, set, nil)))
	assert.Equal(
		t,
		"3h0m0s PROJ-12 Issue 12\n1h0m0s PROJ-13 Issue 13\nTotal: 4h0m0s\n",
		writer.String(),
	)
}

func TestCmdMainInvalidDate(t *testing.T) {
	app, _, set := getBaseAppAndFlagSet("foo")
	assert.Nil(t, set.Parse([]string{"2016-23-25"}))
	assert.EqualError(
		t,
		command.CmdMain(cli.NewContext(app, set, nil)),
		"parsing time \"2016-23-25\": month out of range",
	)
}

func TestCmdMainUsage(t *testing.T) {
	app, _, set := getBaseAppAndFlagSet("foo")
	assert.Nil(t, set.Parse([]string{"1", "2"}))
	assert.EqualError(
		t,
		command.CmdMain(cli.NewContext(app, set, nil)),
		"Usage \"jira-standup {date}\"",
	)
}

func TestCmdMainRelativeDate(t *testing.T) {
	ts := getMockJiraAPI(t, time.Now().Add(-time.Hour*24*20).Format("2006-01-02"))
	defer ts.Close()
	app, writer, set := getBaseAppAndFlagSet(ts.URL)
	assert.Nil(t, set.Parse([]string{"20"}))
	assert.Nil(t, command.CmdMain(cli.NewContext(app, set, nil)))
	assert.Equal(
		t,
		"3h0m0s PROJ-12 Issue 12\n1h0m0s PROJ-13 Issue 13\nTotal: 4h0m0s\n",
		writer.String(),
	)
}

func TestCmdMainAuthError(t *testing.T) {
	ts := getMockJiraAPIAuthError(t)
	defer ts.Close()
	app, _, set := getBaseAppAndFlagSet(ts.URL)
	assert.EqualError(
		t,
		command.CmdMain(cli.NewContext(app, set, nil)),
		"Unable to get client: Auth at JIRA instance failed (HTTP(S) request)."+
			" Request failed. Please analyze the request body for more details. Status code: 403",
	)
}

func TestCmdMainSearchError(t *testing.T) {
	ts := getMockJiraAPISearchError(t, time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
	defer ts.Close()
	app, _, set := getBaseAppAndFlagSet(ts.URL)
	assert.EqualError(
		t,
		command.CmdMain(cli.NewContext(app, set, nil)),
		"Unable to get durations: Unable to make search call: Request failed. Please analyze the request body for more details. Status code: 500",
	)
}

func TestCmdMainInvalidUrl(t *testing.T) {
	app, _, set := getBaseAppAndFlagSet("::/invalid")
	assert.EqualError(t, command.CmdMain(cli.NewContext(app, set, nil)), "Unable to get client: parse ::/invalid: missing protocol scheme")
}

func getMockJiraAPI(t *testing.T, date string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		assert.Nil(t, err)
		r.Body = ioutil.NopCloser(bytes.NewBuffer(b))

		if r.URL.String() == "/rest/auth/1/session" {
			auth := map[string]string{}
			err = json.NewDecoder(r.Body).Decode(&auth)
			assert.Nil(t, err)
			assert.Equal(t, "un", auth["username"])
			assert.Equal(t, "pw", auth["password"])
			http.SetCookie(w, &http.Cookie{Name: "JSESSIONID", Value: "abc"})
			_, err = w.Write([]byte("{}"))
			assert.Nil(t, err)
			return
		}

		if r.URL.String() == fmt.Sprintf(
			"/rest/api/2/search?jql=worklogDate+%%3D+%%27%s%%27+and+worklogAuthor+%%3D+un&startAt=0&maxResults=50&expand=&fields=*all",
			date,
		) {
			session, err := r.Cookie("JSESSIONID")
			assert.Nil(t, err)
			assert.Equal(t, session.Value, "abc")
			today, err := time.Parse("2006-01-02T15:04:05", fmt.Sprintf("%sT12:00:00", date))
			assert.Nil(t, err)
			yesterday := today.Add(-time.Hour * 24)
			tomorrow := today.Add(time.Hour * 24)
			resp := map[string][]interface{}{
				"issues": {
					map[string]interface{}{
						"key": "PROJ-12",
						"fields": map[string]interface{}{
							"summary": "Issue 12",
							"worklog": map[string]interface{}{
								"worklogs": []map[string]interface{}{
									{
										"author": map[string]string{
											"name": "un",
										},
										"created":          today.Format("2006-01-02T15:04:05.999-0700"),
										"timeSpentSeconds": 7200,
									},
									{
										"author": map[string]string{
											"name": "un",
										},
										"created":          today.Format("2006-01-02T15:04:05.999-0700"),
										"timeSpentSeconds": 3600,
									},
									{
										"author": map[string]string{
											"name": "un2",
										},
										"created":          today.Format("2006-01-02T15:04:05.999-0700"),
										"timeSpentSeconds": 3600,
									},
									{
										"author": map[string]string{
											"name": "un",
										},
										"created":          yesterday.Format("2006-01-02T15:04:05.999-0700"),
										"timeSpentSeconds": 7200,
									},
									{
										"author": map[string]string{
											"name": "un",
										},
										"created":          tomorrow.Format("2006-01-02T15:04:05.999-0700"),
										"timeSpentSeconds": 7200,
									},
								},
							},
						},
					},
					map[string]interface{}{
						"key": "PROJ-13",
						"fields": map[string]interface{}{
							"summary": "Issue 13",
							"worklog": map[string]interface{}{
								"worklogs": []map[string]interface{}{
									{
										"author": map[string]string{
											"name": "un",
										},
										"created":          today.Format("2006-01-02T15:04:05.999-0700"),
										"timeSpentSeconds": 3600,
									},
								},
							},
						},
					},
				},
			}
			bytes, _ := json.Marshal(resp)
			_, err = w.Write(bytes)
			assert.Nil(t, err)
			return
		}

		panic(r.URL.String())
	}))
}

func getMockJiraAPIAuthError(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		assert.Nil(t, err)
		r.Body = ioutil.NopCloser(bytes.NewBuffer(b))

		if r.URL.String() == "/rest/auth/1/session" {
			w.WriteHeader(403)
			return
		}

		panic(r.URL.String())
	}))
}

func getMockJiraAPISearchError(t *testing.T, date string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		assert.Nil(t, err)
		r.Body = ioutil.NopCloser(bytes.NewBuffer(b))

		if r.URL.String() == "/rest/auth/1/session" {
			auth := map[string]string{}
			err = json.NewDecoder(r.Body).Decode(&auth)
			assert.Nil(t, err)
			assert.Equal(t, "un", auth["username"])
			assert.Equal(t, "pw", auth["password"])
			http.SetCookie(w, &http.Cookie{Name: "JSESSIONID", Value: "abc"})
			_, err = w.Write([]byte("{}"))
			assert.Nil(t, err)
			return
		}

		if r.URL.String() == fmt.Sprintf(
			"/rest/api/2/search?jql=worklogDate+%%3D+%%27%s%%27+and+worklogAuthor+%%3D+un&startAt=0&maxResults=50&expand=&fields=*all",
			date,
		) {
			w.WriteHeader(500)
			return
		}

		panic(r.URL.String())
	}))
}

func getBaseAppAndFlagSet(mockAPIURL string) (*cli.App, *bytes.Buffer, *flag.FlagSet) {
	set := flag.NewFlagSet("test", 0)
	set.String("url", mockAPIURL, "doc")
	set.String("username", "un", "doc")
	set.String("password", "pw", "doc")
	app := cli.NewApp()
	writer := new(bytes.Buffer)
	app.Writer = writer
	return app, writer, set
}
