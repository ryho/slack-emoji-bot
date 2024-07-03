package main

import (
	"fmt"
	"time"

	"github.com/ryho/slack-emoji-bot/util"
	"github.com/slack-go/slack"
)

func emojisWrapped(allEmojis *SlackEmojiResponseMessage) error {
	// Get the emojis channel
	emojiChannelID, err := getChannel(emojiChannel)
	if err != nil {
		return err
	}
	messages, err := findAllVotePrompts(emojiChannelID)
	if err != nil {
		return err
	}
	for _, msg := range messages {
		timestamp, err := timeFromMessage(msg)
		if err != nil {
			return err
		}
		voters := map[string]util.SetEntry{}
		for _, reaction := range msg.Reactions {
			for _, user := range reaction.Users {
				voters[user] = util.SetEntry{}
			}
		}
		fmt.Printf("Reactions %v, voters %v, date %v\n", len(msg.Reactions), len(voters), timestamp)
	}
	return printTopEmojisByReactionVote(allEmojis, true, 100, messages...)
}

func findAllVotePrompts(emojiChannelId string) ([]*slack.Message, error) {
	conversationParams := &slack.GetConversationHistoryParameters{
		ChannelID: emojiChannelId,
	}
	var reactionMessages []*slack.Message
	for true {
		messages, err := GetConversationHistoryWithBackoff(conversationParams)
		if err != nil {
			return nil, err
		}
		for i, message := range messages.Messages {
			if message.Text == votePrompt || message.Text == votePromptPrevious {
				reactionMessages = append(reactionMessages, &messages.Messages[i])
			}
		}
		if len(messages.ResponseMetaData.NextCursor) == 0 {
			return reactionMessages, nil
		}
		// Check if we have looked through 15 days
		lastMessageTime, err := timeFromMessage(&messages.Messages[len(messages.Messages)-1])
		if err != nil {
			return nil, err
		}
		if time.Since(lastMessageTime) > time.Hour*24*365 {
			return reactionMessages, nil
		}
		conversationParams.Cursor = messages.ResponseMetaData.NextCursor
	}
	return reactionMessages, nil
}
