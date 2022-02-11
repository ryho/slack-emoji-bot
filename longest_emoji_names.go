package main

import "sort"

type StringLengthSort []emoji

func (p StringLengthSort) Len() int           { return len(p) }
func (p StringLengthSort) Less(i, j int) bool { return len(p[i].Name) > len(p[j].Name) }
func (p StringLengthSort) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func longestEmojis(response *SlackEmojiResponseMessage) error {
	sort.Sort(StringLengthSort(response.Emoji))
	message := "Longest Emoji Names:\n"
	for i := 0; i < maxEmojisForLongestEmojis && i < len(response.Emoji); i++ {
		message += printer.Sprintf("%d. :%s: %s (%d)\n", i+1, response.Emoji[i].Name, response.Emoji[i].Name, len(response.Emoji[i].Name))
	}
	_, err := printMessage(MSG_TYPE__PRINT_ONLY, message)
	return err
}
