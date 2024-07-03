package main

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ryho/slack-emoji-bot/util"

	"github.com/slack-go/slack"
)

func dealWithLastWeekMessages() error {
	if overRideLastNewEmoji != "" {
		// If overriding the last new emoji, assume that we are also not
		// able to get last week's votes.
		lastNewEmoji = overRideLastNewEmoji
		previousLastNewEmoji = overRideLastNewEmoji
		return nil
	}
	// Get the emojis channel
	emojiChannelId, err := getChannel(emojiChannel)
	if err != nil {
		return err
	}

	// Get the two messages that we need
	message, lastEmojiMessage, previousLastEmojiMessage, err := findLastWeekMessages(emojiChannelId)
	reactionMessage = message
	if err != nil {
		return err
	}
	// Find the last emoji that was posted last week.
	parts := strings.Split(lastEmojiMessage.Text, ":")
	if len(parts) < 2 {
		return fmt.Errorf("Unable to get last emoji from message %v", lastEmojiMessage)
	}
	lastNewEmoji = parts[len(parts)-2]
	parts = strings.Split(previousLastEmojiMessage.Text, ":")
	if len(parts) < 2 {
		return fmt.Errorf("Unable to get last emoji from message %v", lastEmojiMessage)
	}
	previousLastNewEmoji = parts[len(parts)-2]
	return nil
}

func findLastWeekMessages(emojiChannelId string) (*slack.Message, *slack.Message, *slack.Message, error) {
	conversationParams := &slack.GetConversationHistoryParameters{
		ChannelID: emojiChannelId,
	}
	var reactionMessage, lastEmojiMessage, previousWeekLastEmojiMessage slack.Message
	var foundOne, foundTwo, foundThree, foundFour bool
	for true {
		messages, err := GetConversationHistoryWithBackoff(conversationParams)
		if err != nil {
			return nil, nil, nil, err
		}
		if foundOne && !foundTwo {
			if len(messages.Messages) == 0 {
				return nil, nil, nil, errors.New("Unable to find message " + emojiChannel)
			}
			lastEmojiMessage = messages.Messages[0]
			foundTwo = true
		}
		if foundThree && !foundFour {
			if len(messages.Messages) == 0 {
				return nil, nil, nil, errors.New("Unable to find message " + emojiChannel)
			}
			lastEmojiMessage = messages.Messages[0]
			foundTwo = true
		}
		for i, message := range messages.Messages {
			if message.Text == votePrompt || message.Text == votePromptPrevious {
				if !foundOne {
					reactionMessage = message
					foundOne = true
					if len(messages.Messages) >= i {
						lastEmojiMessage = messages.Messages[i+1]
						foundTwo = true
					}
				} else {
					previousWeekLastEmojiMessage = message
					foundThree = true
					if len(messages.Messages) >= i {
						previousWeekLastEmojiMessage = messages.Messages[i+1]
						foundFour = true
						break
					}
				}
			}
		}
		if foundFour {
			break
		}
		if len(messages.ResponseMetaData.NextCursor) == 0 {
			return nil, nil, nil, errors.New("Unable to find message in channel " + emojiChannel)
		}
		// Check if we have looked through 15 days
		lastMessageTime, err := timeFromMessage(&messages.Messages[len(messages.Messages)-1])
		if err != nil {
			return nil, nil, nil, err
		}
		if time.Since(lastMessageTime) > time.Hour*24*16 {
			return nil, nil, nil, errors.New("Unable to find message in channel " + emojiChannel + " in the last 15 days")
		}
		conversationParams.Cursor = messages.ResponseMetaData.NextCursor
	}
	return &reactionMessage, &lastEmojiMessage, &previousWeekLastEmojiMessage, nil
}

func timeFromMessage(message *slack.Message) (time.Time, error) {
	seconds, err := strconv.ParseFloat(message.Timestamp, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, int64(float64(time.Second)*seconds)), nil
}

func getChannel(channelName string) (string, error) {
	if len(channelName) == 0 {
		return "", errors.New("No channel name provided")
	}
	if cachedChannelID != "" {
		return cachedChannelID, nil
	}
	if channelName[0] == '#' {
		channelName = channelName[1:]
	}
	channelsParams := &slack.GetConversationsParameters{
		ExcludeArchived: true,
		Limit:           1000,
	}
	var emojiChannelData slack.Channel
	for true {
		channels, cursor, err := GetConversationsWithBackoff(channelsParams)
		if err != nil {
			return "", err
		}
		var found bool
		for _, channel := range channels {
			if channel.IsChannel && channel.Name == channelName {
				found = true
				emojiChannelData = channel
				break
			}
		}
		if found {
			break
		}
		if cursor == "" {
			return "", errors.New("Unable to find channel " + emojiChannel)
		}
		channelsParams.Cursor = cursor
	}
	fmt.Printf("Found Channel ID: %v. You can set this in cachedChannelID in config.go for faster run times.\n", emojiChannelData.ID)
	return emojiChannelData.ID, nil
}

func printTopEmojisByReactionVote(allEmojis *SlackEmojiResponseMessage, doEmojisWrapped bool, maxPrintCount int, messages ...*slack.Message) error {
	var emojis []*stringCount
	uniqueUsers := util.StringSet{}
	for _, message := range messages {
		for _, reaction := range message.Reactions {
			emojis = append(emojis, &stringCount{name: reaction.Name, count: reaction.Count})
			for _, user := range reaction.Users {
				uniqueUsers[user] = util.SetEntry{}
			}
		}
	}
	sort.Sort(ByCount(emojis))

	printedCount := 0
	previousCount := math.MaxInt64

	minReaction := 3
	var creators []string
	var counts []int
	var printedEmojis []string
	for _, emoji := range emojis {
		// Stop if we have printed enough emojis, however, always print all emojis with the same reaction
		// count even if we go over the limit.
		if emoji.count != previousCount && printedCount >= maxPrintCount {
			break
		}
		// Stop if the reaction count is too low, even if we have not hit the limit.
		if emoji.count < minReaction {
			break
		}
		emojisObj := allEmojis.emojiMap[emoji.name]
		creators = append(creators, emojisObj.UserId)
		counts = append(counts, emoji.count)
		var name string
		if aprilFoolsMode {
			name = aprilFoolsEmoji
		} else {
			name = emoji.name
		}
		printedEmojis = append(printedEmojis, name)
		previousCount = emoji.count
		printedCount++
	}

	peopleToPrint := TopPeopleToPrint
	message := fmt.Sprintf(lastWeek, len(uniqueUsers))
	if doEmojisWrapped {
		peopleToPrint = 20
		rightNow := time.Now()
		year := rightNow.Year()
		if rightNow.Month() == time.January {
			year--
		}
		message = fmt.Sprintf(lastYear, year, len(uniqueUsers))
	}

	return printTopCreators(message, peopleToPrint, creators, counts, printedEmojis)
}
