package main

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/slack-go/slack"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// START of things you should edit
// See config.go for more things to edit
const (
	// Caching all emoji images can take a lot of time and storage, so you
	// may want to leave it off. It takes 547MB for my company's Slack.
	// The only use of cashing all images is to see what deleted emojis were.
	// After all emojis have been downloaded the first time, this will only download new emojis.
	// You can also download the image for a deleted emoji if it has not been deleted for very long.
	cacheImages = true

	// This controls if the JSON blob of all current emojis is cached. This is only used for detecting
	// deleted emojis.
	cacheEmojiDumps = true

	// Top uploaders of all time is noisy.
	// I only send at the end of the year, if someone has recently moved up a lot, etc.
	sendTopUploadersAllTime = false

	// The channel to post these messages in when in FULL_SEND mode.
	emojiChannel = "#emojis"

	// This controls if things are printed, DMed or posted publicly.
	// See values below.
	runMode Mode = MODE__DM_FOR_TESTING
)

// END of things you should edit
// See config.go for more things to edit

type Mode int

const (
	MODE__PRINT_EVERYTHING Mode = iota
	MODE__DM_FOR_REVIEW
	MODE__DM_FOR_TESTING
	MODE__FULL_SEND
)

func main() {
	slackApi = slack.New(botOauthToken)

	allEmojis, err := getAllEmojis()
	if err != nil {
		panic(err)
	}

	// This will get the last new emoji, and print the top emojis reactions.
	err = dealWithLastWeekMessages(allEmojis)
	if err != nil {
		panic(err)
	}

	// cacheEmojiImages and detectDeletedEmojis should be called before removeSkippedEmojis
	err = cacheEmojiImages(allEmojis)
	if err != nil {
		panic(err)
	}

	err = detectDeletedEmojis(allEmojis)
	if err != nil {
		panic(err)
	}

	removeSkippedEmojis(allEmojis)

	// mostRecentEmojis, topUploaders, and longestEmojis should be called after removeSkippedEmojis
	err = mostRecentEmojis(allEmojis)
	if err != nil {
		panic(err)
	}

	err = topUploaders(allEmojis)
	if err != nil {
		panic(err)
	}

	err = longestEmojis(allEmojis)
	if err != nil {
		panic(err)
	}
}

var (
	slackApi     *slack.Client
	printer      = message.NewPrinter(language.English)
	lastNewEmoji string
)

const (
	maxEmojisPerMessage       = 22
	maxPeopleForTopUploaders  = 100
	maxEmojisForLongestEmojis = 100
	maxCharactersPerMessage   = 10000
	TopPeopleToPrint          = 5

	lastWeek           = "Congratulations to the top emojis from last week (sorted by emoji reactions):\n"
	introMessage       = "Here are all the new emojis! There are %v new emojis from %v people."
	lastMessage        = "React here with the best new emojis!"
	topAllTimeMessage  = "Top Emoji Uploaders of All Time:"
	topThisWeekMessage = "Top Emoji Uploaders This Week:"
	muteMessage        = "If you do not want to be pinged by this list, message @%s to request that you be added to the mute list so the script prints your name without the @ sign.\n"
	skipMessage        = "If you want to be excluded from the list all together, you can ask @%s to add you to the skip list.\n"
)

func mostRecentEmojis(response *SlackEmojiResponseMessage) error {
	var allNewEmojis []string
	people := map[string]*stringCount{}
	var foundLastEmoji bool
	lastNewEmojiSanitized := strings.ReplaceAll(lastNewEmoji, ":", "")

	sort.Sort(EmojiUploadDateSort(response.Emoji))

	for _, emoji := range response.Emoji {
		if emoji.Name == lastNewEmojiSanitized {
			foundLastEmoji = true
			break
		}
		count, ok := people[emoji.UserId]
		if !ok {
			people[emoji.UserId] = &stringCount{
				name:  emoji.UserDisplayName,
				id:    emoji.UserId,
				count: 1,
			}
		} else {
			count.count++
		}
		allNewEmojis = append(allNewEmojis, emoji.Name)
	}
	if !foundLastEmoji {
		fmt.Printf("Did not find the last emoji %v. This is probably a problem.\n", lastNewEmoji)
	}

	slackMessages := make([]string, 0, len(response.Emoji)/maxEmojisPerMessage+1)
	i := 0
	j := 0
	auditMessage := []string{""}
	for z := len(allNewEmojis) - 1; z >= 0; z-- {
		emojiName := allNewEmojis[z]
		if emojiName == lastNewEmojiSanitized {
			foundLastEmoji = true
			break
		}
		newPart := ":" + emojiName + ": " + emojiName + "\n"
		if len(newPart)+len(auditMessage[len(auditMessage)-1]) > maxCharactersPerMessage {
			auditMessage = append(auditMessage, newPart)
		} else {
			auditMessage[len(auditMessage)-1] += newPart
		}
		if len(slackMessages) > i {
			slackMessages[i] = slackMessages[i] + ":" + emojiName + ":"
		} else {
			slackMessages = append(slackMessages, ":"+emojiName+":")
		}
		if j == maxEmojisPerMessage {
			i++
			j = 0
		} else {
			j++
		}

	}

	_, err := printMessage(MSG_TYPE__SEND, printer.Sprintf(introMessage, len(allNewEmojis), len(people)))
	if err != nil {
		return err
	}

	for _, msg := range slackMessages {
		_, err := printMessage(MSG_TYPE__SEND_AND_REVIEW, msg)
		if err != nil {
			return err
		}
	}
	var peopleNameArray []string
	for _, person := range people {
		peopleNameArray = append(peopleNameArray, person.name)
	}

	threadId, err := printMessage(MSG_TYPE__SEND, lastMessage)
	if err != nil {
		return err
	}
	// Side by side message
	for _, part := range auditMessage {
		_, err := printMessageWithThreadId(MSG_TYPE__SEND, part, threadId)
		if err != nil {
			return err
		}
	}

	// List everyone's names
	_, err = printMessageWithThreadId(MSG_TYPE__SEND, createNameString(peopleNameArray), threadId)
	if err != nil {
		return err
	}

	return printTopPeople(topThisWeekMessage, people, math.MaxInt64, false)
}

type EmojiUploadDateSort []emoji

func (p EmojiUploadDateSort) Len() int           { return len(p) }
func (p EmojiUploadDateSort) Less(i, j int) bool { return p[i].Created > p[j].Created }
func (p EmojiUploadDateSort) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type ByCount []*stringCount

func (a ByCount) Len() int { return len(a) }
func (a ByCount) Less(i, j int) bool {
	if a[i].count == a[j].count {
		return a[i].name < a[j].name
	}
	return a[i].count > a[j].count
}
func (a ByCount) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type stringCount struct {
	count int
	name  string
	id    string
}
