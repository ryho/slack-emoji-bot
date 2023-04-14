package main

import (
	"math/rand"
	"strings"
)

func bringsYouCounter(response *SlackEmojiResponseMessage) error {
	var bufoEmojis []string
	var bringsYouEmojis []string
	for _, emoji := range response.Emoji {
		if strings.Contains(emoji.Name, "bringsyou") ||
			strings.Contains(emoji.Name, "brings-you") ||
			strings.Contains(emoji.Name, "brings_you") ||
			strings.Contains(emoji.Name, "bringyou") ||
			strings.Contains(emoji.Name, "bring-you") ||
			strings.Contains(emoji.Name, "bring_you") ||
			strings.Contains(emoji.Name, "he-bringin") {
			// increment the counter
			bringsYouEmojis = append(bringsYouEmojis, emoji.Name)
		}
		if strings.Contains(emoji.Name, "bufo") || strings.Contains(emoji.Name, "froge") {
			bufoEmojis = append(bufoEmojis, emoji.Name)
		}
	}
	var startEmoji string
	if len(bufoEmojis) > len(bringsYouEmojis) {
		startEmoji = ":bufo-appears:"
	} else {
		startEmoji = ":he-brings-you-metrics:"
	}
	randomBufoEmoji := bufoEmojis[rand.Intn(len(bufoEmojis))]
	randomBringsYouEmoji := bringsYouEmojis[rand.Intn(len(bringsYouEmojis))]
	_, err := printMessage(MSG_TYPE__SEND_AND_REVIEW,
		printer.Sprintf("%s There are now %d *he-brings-you* emojis :%s: and %d *Bufo* emojis :%s:!",
			startEmoji, len(bringsYouEmojis), randomBringsYouEmoji, len(bufoEmojis), randomBufoEmoji))

	return err
}
