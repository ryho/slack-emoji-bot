package main

import "strings"

func removeSkippedEmojis(response *SlackEmojiResponseMessage) {
	// Remove colons. This allows the emojis to be specified as
	// :emoji_name: or just emoji_name
	for emoji := range skipEmojis {
		strings.ReplaceAll(emoji, ":", "")
	}

	for i := 0; i < len(response.Emoji); i++ {
		emoji := response.Emoji[i]
		if _, ok := skipEmojis[emoji.Name]; ok {
			response.Emoji[i] = response.Emoji[len(response.Emoji)-1]
			response.Emoji = response.Emoji[:len(response.Emoji)-1]
		}
	}
}
