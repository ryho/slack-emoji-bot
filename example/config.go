package main

import "github.com/ryho/slack-emoji-bot/util"

// Copy this to the parent directory before filling in!
// The gitignore will prevent this from being sent to GitHUb

// The Slack user name of the person who should be contacted in case of problems.
// This should not include the @ symbol.
const ownerLDAP = "TODO"

// Should look like U0XXXXXXXX
// TODO: Explain how to get this
const ownerUserId = "TODO"

var additionalReviewerIds = []string{}

// TODO: Explain how to get this
const botOauthToken = "TODO"

// TODO: Explain how to get this
const ownerUserOauthToken = `TODO`

// TODO: Explain how to get this
const ownerUserCookie = ``

// People really dislike pictures of some frog.
// Sometimes you have to keep the peace...
// Can be specified with or without the colons.
var skipEmojis = util.StringSet{
	"TODO": {},
}

// Some people prefer not to be pinged to join the channel.
var muteLDAPs = util.StringSet{
	"TODO": {},
}

// Some people may not want to be a part of this at all.
var skipLDAPs = util.StringSet{
	"TODO": {},
}
