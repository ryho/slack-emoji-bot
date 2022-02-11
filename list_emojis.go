package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	emojiListUrl = "https://square.slack.com/api/emoji.adminList"
)

type SlackEmojiResponseMessage struct {
	Ok       bool    `json:"ok"`
	Emoji    []emoji `json:"emoji"`
	emojiMap map[string]emoji
}

type emoji struct {
	Name            string
	IsAlias         int    `json:"is_alias"`
	AliasFor        string `json:"alias_for"`
	Url             string
	Created         int
	TeamId          string `json:"team_id"`
	UserId          string `json:"user_id"`
	UserDisplayName string `json:"user_display_name"`
}

func getAllEmojis() (*SlackEmojiResponseMessage, error) {
	commandResponse, err := getEmojis()
	if err != nil {
		return nil, err
	}
	if cacheEmojiDumps {
		err := cacheEmojiResponse(commandResponse)
		if err != nil {
			return nil, err
		}
	}

	allEmojis, err := parseEmojiResponse(commandResponse)
	if err != nil {
		return nil, err
	}
	allEmojis.emojiMap = make(map[string]emoji, len(allEmojis.Emoji))
	for i, emoji := range allEmojis.Emoji {
		allEmojis.emojiMap[emoji.Name] = allEmojis.Emoji[i]
	}
	return allEmojis, nil
}

func getEmojis() ([]byte, error) {
	vals := url.Values{}
	vals.Set("token", ownerUserOauthToken)
	vals.Set("page", "1")
	vals.Set("count", "100000000")
	vals.Set("sort_by", "created")
	vals.Set("sort_dir", "desc")
	vals.Set("_x_mode", "online")
	resp, err := http.PostForm(emojiListUrl, vals)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bodyBytes, nil
}

func parseEmojiResponse(response []byte) (*SlackEmojiResponseMessage, error) {
	var responseParsed SlackEmojiResponseMessage
	err := json.Unmarshal(response, &responseParsed)
	if err != nil {
		return nil, err
	}
	return &responseParsed, nil
}
