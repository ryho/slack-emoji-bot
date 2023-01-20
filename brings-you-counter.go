package main

import (
	"strconv"
	"strings"
)

func bringsYouCounter(response *SlackEmojiResponseMessage) error {
	bringsYouCounter := 0
	for _, emoji := range response.Emoji {
		if strings.Contains(emoji.Name, "bringsyou") ||
			strings.Contains(emoji.Name, "brings-you") ||
			strings.Contains(emoji.Name, "brings_you") ||
			strings.Contains(emoji.Name, "bringyou") ||
			strings.Contains(emoji.Name, "bring-you") ||
			strings.Contains(emoji.Name, "bring_you") ||
			strings.Contains(emoji.Name, "he-bringin") {
			// increment the counter
			bringsYouCounter++
		}
	}
	_, err := printMessage(MSG_TYPE__SEND_AND_REVIEW, ":he-brings-you-metrics: There are now "+strconv.Itoa(bringsYouCounter)+" *he-brings-you* emojis!")
	return err
}
