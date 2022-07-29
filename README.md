# Slack Emoji Bot

This Slack bot will post the newly created custom Slack emojis to the channel of your choice.

Features:
- Prints the Slack emojis uploaded in the last week.
- Interactive vote for most popular emoji of the week. Winner is posted the next week.
- Top uploaders of the week by count.
- Top uploaders of all time by count.
- Longest emoji names.
- Detection of deleted emojis.
- Caches all emoji images.
- April fools mode to send all emojis as a broken image emoji.

TODO:
- Welcome people that joined in the past week.
- Detect people's first uploaded emoji and congratulate.
- Rework how settings are configured. Currently, they are hardcoded in a Golang file.
- Send deleted emojis in "DM for review" mode.
- Improve behavior when this is your first time running the script.
  - It would post all emojis ever if you don't specify an emoji from 7 days ago which is tricky to get if you have not run this before.
  - It should support skipping "top emojis from last week" if this is the first week.
- Bug: If someone creates an alias, and people vote for that alias, the next week the person who
uploaded the original emoji will show up on the most popular emoji ranking. Not sure if this can be fixed. 
- Make this program be a running process, not a script. This will unlock many more features.
  - Automate users asking to be muted or skipped.
Backlog (lol)
- Stop using undocumented endpoint for fetching emojis, use the real API.