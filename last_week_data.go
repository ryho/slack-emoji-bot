package main

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

func dealWithLastWeekMessages(allEmojis *SlackEmojiResponseMessage) error {
	if overRideLastNewEmoji != "" {
		// If overriding the last new emoji, assume that we are also not
		// able to get last week's votes.
		lastNewEmoji = overRideLastNewEmoji
		return nil
	}
	// Get the emojis channel
	emojiChannelData, err := getChannel(emojiChannel)
	if err != nil {
		return err
	}

	// Get the two messages that we need
	reactionMessage, lastEmojiMessage, err := findLastWeekMessages(emojiChannelData)
	if err != nil {
		return err
	}
	err = printTopEmojisByReactionVote(reactionMessage, allEmojis)
	if err != nil {
		return err
	}
	// Find the last emoji that was posted last week.
	parts := strings.Split(lastEmojiMessage.Text, ":")
	if len(parts) < 2 {
		return fmt.Errorf("Unable to get last emoji from message %v", lastEmojiMessage)
	}
	lastNewEmoji = parts[len(parts)-2]
	return nil
}

func findLastWeekMessages(emojiChannelData *slack.Channel) (*slack.Message, *slack.Message, error) {
	conversationParams := &slack.GetConversationHistoryParameters{
		ChannelID: emojiChannelData.ID,
	}
	var reactionMessage, lastEmojiMessage slack.Message
	var foundOne, foundBoth bool
	for true {
		messages, err := slackApi.GetConversationHistory(conversationParams)
		if err != nil {
			return nil, nil, err
		}
		if foundOne {
			if len(messages.Messages) == 0 {
				return nil, nil, errors.New("Unable to find message " + emojiChannel)
			}
			lastEmojiMessage = messages.Messages[0]
			foundBoth = true
			break
		}
		for i, message := range messages.Messages {
			if message.Text == lastMessage {
				reactionMessage = message
				foundOne = true
				if len(messages.Messages) >= i {
					lastEmojiMessage = messages.Messages[i+1]
					foundBoth = true
					break
				}
			}
		}
		if foundBoth {
			break
		}
		if len(messages.ResponseMetadata.Cursor) == 0 {
			return nil, nil, errors.New("Unable to find message in channel " + emojiChannel)
		}
		// Check if we have looked through 15 days
		seconds, err := strconv.ParseFloat(messages.Messages[len(messages.Messages)-1].Timestamp, 64)
		if err != nil {
			return nil, nil, err
		}
		lastMessageTime := time.Unix(0, int64(float64(time.Second)*seconds))
		if time.Since(lastMessageTime) > time.Hour*24*15 {
			return nil, nil, errors.New("Unable to find message in channel " + emojiChannel + " in the last 15 days")
		}
		conversationParams.Cursor = messages.ResponseMetadata.Cursor
	}
	return &reactionMessage, &lastEmojiMessage, nil
}

func getChannel(channelName string) (*slack.Channel, error) {
	if len(channelName) == 0 {
		return nil, errors.New("No channel name provided")
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
		channels, cursor, err := slackApi.GetConversations(channelsParams)
		if err != nil {
			return nil, err
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
			return nil, errors.New("Unable to find channel " + emojiChannel)
		}
		channelsParams.Cursor = cursor
	}
	return &emojiChannelData, nil
}

// This takes in the copied text of a message from Slack, and prints the top Emoji reactions.
func printTopEmojisByReactionVote(message *slack.Message, allEmojis *SlackEmojiResponseMessage) error {
	var emojis []*stringCount
	uniqueUsers := map[string]struct{}{}
	for _, reaction := range message.Reactions {
		emojis = append(emojis, &stringCount{name: reaction.Name, count: reaction.Count})
		for _, user := range reaction.Users {
			uniqueUsers[user] = struct{}{}
		}
	}
	sort.Sort(ByCount(emojis))

	printedCount := 0
	previousCount := math.MaxInt64

	maxPrintCount := 10
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

	return printTopCreators(fmt.Sprintf(lastWeek, len(uniqueUsers)), creators, counts, printedEmojis)
}
