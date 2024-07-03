package main

import (
	"math/rand"
	"strings"
)

func bringsYouCounter(response *SlackEmojiResponseMessage) error {
	var bufoEmojis []string
	var bringsYouEmojis []string

	lastNewEmojiSanitized := strings.ReplaceAll(lastNewEmoji, ":", "")
	for _, emoji := range response.Emoji {
		if fastMode && emoji.Name == lastNewEmojiSanitized {
			break
		}
		if strings.Contains(emoji.Name, "bringsyou") ||
			strings.Contains(emoji.Name, "brings-you") ||
			strings.Contains(emoji.Name, "brings_you") ||
			strings.Contains(emoji.Name, "bringyou") ||
			strings.Contains(emoji.Name, "bring-you") ||
			strings.Contains(emoji.Name, "bring_you") ||
			strings.Contains(emoji.Name, "he-bringin") ||
			strings.Contains(emoji.Name, "hbu-") {
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
	var err error
	if fastMode {
		_, err = printMessage(MSG_TYPE__SEND_AND_REVIEW,
			printer.Sprintf("%s There are %d new *he-brings-you* emojis :%s: and %d new *Bufo* emojis :%s: this week!",
				startEmoji, len(bringsYouEmojis), randomBringsYouEmoji, len(bufoEmojis), randomBufoEmoji))
	} else {
		_, err = printMessage(MSG_TYPE__SEND_AND_REVIEW,
			printer.Sprintf("%s There are now %d *he-brings-you* emojis :%s: and %d *Bufo* emojis :%s:!",
				startEmoji, len(bringsYouEmojis), randomBringsYouEmoji, len(bufoEmojis), randomBufoEmoji))
	}

	return err
}
