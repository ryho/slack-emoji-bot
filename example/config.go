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

// TODO: Explain how to get this
const botOauthToken = "TODO"

// TODO: Explain how to get this
const ownerUserOauthToken = `TODO`

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
