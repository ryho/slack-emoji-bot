package main

// Copy this to the parent directory before filling in!
// The gitignore will prevent this from being sent to GitHUb

// Put the last emoji from the last post here. Can be specified with or without the colons.
const lastNewEmoji = ":TODO:"

// Copy last week's "React here with the best new emojis!" message along with all the emoji reactions.
// It is ok if you are sloppy about what you copy, it will still be handled correctly.
const lastWeekReactions = "React here with the best TODO"

// The Slack user name of the person who should be contacted in case of problems.
// This should not include the @ symbol.
const ownerLDAP = "TODO"

// Should look like U0XXXXXXXX
const ownerUserId = "TODO"

var additionalReviewerIds = []string{}

// Visit https://square.slack.com/customize/emoji in Chrome.
// Open the console, go to the network tab.
// Sort by date on the emoji page, copy the network request to the emoji url as curl,
// and paste that here
// It should look like
// curl 'https://square.slack.com/api/emoji.adminList?_x_id=bla \
// -H 'bla'
// ...
const emojiCurl = `TODO`

// Look for a request to https://edgeapi.slack.com/cache/ABC/XYZ/users/info and copy as curl as well.
// Do this to the updated_ids part:
// "updated_ids":{INSERT_USER_IDS_HERE}}'
const userCurl = `TODO`

const oauthToken = "TODO"
const emojiChannel = "#emojis"

// People really dislike pictures of some frog.
// Sometimes you have to keep the peace...
// Can be specified with or without the colons.
var skipEmojis = map[string]struct{}{
	"TODO": {},
}

// Some people prefer not to be pinged to join the channel.
var muteLDAPs = map[string]struct{}{
	"TODO": {},
}

// Some people may not want to be a part of this at all.
var skipLDAPs = map[string]struct{}{
	"TODO": {},
}
