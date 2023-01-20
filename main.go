package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

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

	findLongestEmojisAllTime = false

	skipTopEmojisByReactionVote = false

	// If this is turned on, emojis in the format "emoji-name-123" will be skipped.
	// This is the format used by Slack if another work space's emojis are merged in and there are duplicate names.
	skipDuplicateBulkImportEmojis = false
	// Do not include emojis if after removing - and _ they are a dupe of an existing emoji.
	strictUniqueMode = false
	// Skip emojis that begin with screen-shot-, the format that Mac uses by default.
	skipScreenShots = true
	// Disables the emojis to skip list.
	literally1984Mode = true

	// Replaces all emoijs sent with a single emoji, ideally an old school windows broken image emoji.
	aprilFoolsMode = false
	// Do not include the colons
	aprilFoolsEmoji = "broken-img"

	// Needs to be used after turning off April Fools mode
	// to prevent the joke emoji from being detected as the last mew emoji.
	// Can also be used when running the script for the first time.
	// Can have colons or not, doesn't matter.
	overRideLastNewEmoji = ""

	// The channel to post these messages in when in FULL_SEND mode.
	emojiChannel = "#emojis"

	// This controls if things are printed, DMed or posted publicly.
	// See values below.
	runMode Mode = MODE__FULL_SEND

	doEmojisWrapped      = false
	doHeBringsYouCounter = true
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
	start := time.Now()
	defer func() {
		fmt.Printf("Time spent: %v\n", time.Since(start))
	}()
	slackApi = slack.New(botOauthToken)

	allEmojis, err := getAllEmojis()
	if err != nil {
		panic(err)
	}

	if doEmojisWrapped {
		err = emojisWrapped(allEmojis)
		if err != nil {
			panic(err)
		}
		return
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

	err = topAndNewUploaders(allEmojis)
	if err != nil {
		panic(err)
	}

	if doHeBringsYouCounter {
		err = bringsYouCounter(allEmojis)
		if err != nil {
			panic(err)
		}
	}

	if findLongestEmojisAllTime {
		err = longestEmojis(allEmojis)
		if err != nil {
			panic(err)
		}
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

	lastWeek                  = ":trophy: *Congratulations* to the top new emojis from last week (sorted by emoji reactions from %v voters):\n"
	lastYear                  = ":trophy::trophy::trophy: *CONGRATULATIONS TO THE TOP EMOJIS OF 2022!!!* (sorted by emoji reactions from %v voters):\n"
	introMessage              = ":new-shine: Here are all the new emojis! There are %v new emojis from %v people."
	votePrompt                = ":votesticker: *Vote for the best new emoji of the week by reacting here!*"
	votePromptPrevious        = "Vote for the best new emoji of the week by reacting here!"
	topAllTimeMessage         = ":tophat: Top Emoji Uploaders of All Time:"
	topThisWeekMessage        = ":rocket: Top Emoji Uploaders This Week:"
	topSecondMessage          = "More Top Emoji Uploaders:"
	newUploadersMessage       = ":welcome: *Welcome* to %d New Emoji Uploaders!"
	newUploadersSecondMessage = "More New Emoji Uploaders:"
	muteMessage               = "If you do not want to be pinged by this bot, message @%s to request that you be added to the mute list so the script prints your name without the @ sign.\n"
	skipMessage               = "If you want to be excluded from the bot all together, you can ask @%s to add you to the skip list.\n"
)

func mostRecentEmojis(response *SlackEmojiResponseMessage) error {
	var allNewEmojis []string
	response.peopleThisWeek = map[string]*stringCount{}
	var foundLastEmoji bool
	lastNewEmojiSanitized := strings.ReplaceAll(lastNewEmoji, ":", "")

	sort.Sort(EmojiUploadDateSort(response.Emoji))

	for _, emoji := range response.Emoji {
		if emoji.Name == lastNewEmojiSanitized {
			foundLastEmoji = true
			break
		}
		count, ok := response.peopleThisWeek[emoji.UserId]
		if !ok {
			response.peopleThisWeek[emoji.UserId] = &stringCount{
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
		if aprilFoolsMode {
			emojiName = aprilFoolsEmoji
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

	_, err := printMessage(MSG_TYPE__SEND_AND_REVIEW, printer.Sprintf(introMessage, len(allNewEmojis), len(response.peopleThisWeek)))
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
	for _, person := range response.peopleThisWeek {
		peopleNameArray = append(peopleNameArray, person.name)
	}

	threadId, err := printMessage(MSG_TYPE__SEND, votePrompt)
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

	return printTopPeople(topThisWeekMessage, topSecondMessage, response.peopleThisWeek, math.MaxInt64, false)
}

type EmojiUploadDateSort []*emoji

func (p EmojiUploadDateSort) Len() int           { return len(p) }
func (p EmojiUploadDateSort) Less(i, j int) bool { return p[i].Created > p[j].Created }
func (p EmojiUploadDateSort) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type EmojiUploadDateSortBackwards []*emoji

func (p EmojiUploadDateSortBackwards) Len() int           { return len(p) }
func (p EmojiUploadDateSortBackwards) Less(i, j int) bool { return p[i].Created < p[j].Created }
func (p EmojiUploadDateSortBackwards) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

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
