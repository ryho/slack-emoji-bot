package main

import (
	"math/rand"
	"strings"
)

type EmojiMeme struct {
	EmojiName string

	SubStrings  []string
	StartEmoji  string
	NoNewEmojis string
}

func memeCounter(response *SlackEmojiResponseMessage) error {
	if len(EmojiMemes) == 0 {
		return nil
	}

	lastNewEmojiSanitized := strings.ReplaceAll(lastNewEmoji, ":", "")
	newMemeEmojis := make([][]string, len(EmojiMemes))

	// Find all emojis that are part of the given meme type
	for _, emoji := range response.Emoji {
		if emoji.Name == lastNewEmojiSanitized {
			break
		}
		for i, meme := range EmojiMemes {
			for _, subString := range meme.SubStrings {
				if strings.Contains(emoji.Name, subString) {
					newMemeEmojis[i] = append(newMemeEmojis[i], emoji.Name)
					break
				}
			}
		}
	}
	// Choose a start emoji from the emoji type with the most new emojis
	startEmoji := genericSadEmoji
	var maxNewEmojis int
	for i, emojiMeme := range EmojiMemes {
		if len(newMemeEmojis[i]) > maxNewEmojis {
			maxNewEmojis = len(newMemeEmojis[i])
			startEmoji = emojiMeme.StartEmoji
		}
	}

	// Choose random emojis for the meme types
	var randomMemeEmojis []string
	for i, emojiMeme := range EmojiMemes {
		if len(newMemeEmojis[i]) == 0 {
			randomMemeEmojis = append(randomMemeEmojis, emojiMeme.NoNewEmojis)
		} else {
			randomMemeEmojis = append(randomMemeEmojis, newMemeEmojis[i][rand.Intn(len(newMemeEmojis[i]))])
		}
	}

	var messages []string
	for i, emojiMeme := range EmojiMemes {
		messages = append(messages, printer.Sprintf("%d new *%s* emojis :%s:", len(newMemeEmojis[i]), emojiMeme.EmojiName, randomMemeEmojis[i]))
	}
	messages[len(messages)-1] = "and " + messages[len(messages)-1]

	_, err := printMessage(MSG_TYPE__SEND_AND_REVIEW,
		printer.Sprintf(":%s: There are %s this week!",
			startEmoji, strings.Join(messages, ", ")))
	return err
}
