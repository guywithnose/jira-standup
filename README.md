**jira-standup** summarizes the tickets that you have recently tracked time against.

### Example Usage
```bash
jira-standup --username bob --password $uP3rS3cReT --url https://jira.example.com:8000
3h0m0s PROJ-12 Issue 12
1h0m0s PROJ-13 Issue 13
Total: 4h0m0s",
```

### Check yesterday
```bash
jira-standup --username bob --password $uP3rS3cReT \
--url https://jira.example.com:8000 --relativeDate 1
```

### Check specific date
```bash
jira-standup --username bob --password $uP3rS3cReT \
--url https://jira.example.com:8000 --date 2017-10-25
```
