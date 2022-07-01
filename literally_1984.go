package main

import (
	"sort"
	"strings"

	"github.com/ryho/slack-emoji-bot/util"
)

func removeSkippedEmojis(response *SlackEmojiResponseMessage) {
	// Remove colons. This allows the emojis to be specified as
	// :emoji_name: or just emoji_name
	for emoji := range skipEmojis {
		if emoji[0] == ':' {
			skipEmojis[strings.ReplaceAll(emoji, ":", "")] = util.SetEntry{}
			delete(skipEmojis, emoji)
		}
	}
	uniqueNames := util.StringSet{}

	sort.Sort(EmojiUploadDateSortBackwards(response.Emoji))
	var newEmojiList []*emoji
	for i := 0; i < len(response.Emoji); i++ {
		emoji := response.Emoji[i]
		if _, ok := skipEmojis[emoji.Name]; ok {
			delete(response.emojiMap, emoji.Name)
			continue
		}
		if skipScreenShots && strings.HasPrefix(emoji.Name, "screen-shot-") {
			delete(response.emojiMap, emoji.Name)
			continue
		}
		if skipDuplicateBulkImportEmojis {
			// If this is turned on, emojis in the format "emoji-name-123" will be skipped.
			// This is the format used by Slack if another work space's emojis are merged in and there are duplicate names.
			lastOccurrence := strings.LastIndex(emoji.Name, "-")
			if lastOccurrence != -1 {
				ending := emoji.Name[lastOccurrence+1:]
				if len(ending) > 0 && onlyNumbers(ending) {
					delete(response.emojiMap, emoji.Name)
					continue
				}
			}
		}
		// Do not include emojis if after removing - and _ they are a dupe of an existing emoji.
		if strictUniqueMode {
			cleanName := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(emoji.Name), "-", ""), "_", "")
			if _, ok := uniqueNames[cleanName]; ok {
				delete(response.emojiMap, emoji.Name)
				continue
			}
			uniqueNames[cleanName] = util.SetEntry{}
		}
		newEmojiList = append(newEmojiList, emoji)
	}
	response.Emoji = newEmojiList
}

func onlyNumbers(input string) bool {
	for _, character := range input {
		if !(character >= '0' && character <= '9') {
			return false
		}
	}
	return true
}
