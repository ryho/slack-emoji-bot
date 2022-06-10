package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

const (
	emojiListUrl = "https://square.slack.com/api/emoji.adminList"
	pageSize     = 20000
)

type SlackEmojiResponseMessage struct {
	Ok                    bool           `json:"ok"`
	Emoji                 []*emoji       `json:"emoji"`
	CustomEmojiTotalCount int64          `json:"custom_emoji_total_count"`
	Paging                PagingResponse `json:"paging"`
	emojiMap              map[string]*emoji
}

type PagingResponse struct {
	Count int `json:"count"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Pages int `json:"pages"`
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
	var allEmojis, currentPage *SlackEmojiResponseMessage
	for page := 1; currentPage == nil || page < currentPage.Paging.Pages; page++ {
		commandResponse, err := getEmojis(page)
		if err != nil {
			return nil, err
		}
		currentPage, err = parseEmojiResponse(commandResponse)
		if err != nil {
			return nil, err
		}
		if allEmojis == nil {
			allEmojis = currentPage
		} else {
			allEmojis.Emoji = append(allEmojis.Emoji, currentPage.Emoji...)
		}
	}
	if cacheEmojiDumps {
		err := cacheEmojiResponse(allEmojis)
		if err != nil {
			return nil, err
		}
	}

	allEmojis.emojiMap = make(map[string]*emoji, len(allEmojis.Emoji))
	for i, emoji := range allEmojis.Emoji {
		allEmojis.emojiMap[emoji.Name] = allEmojis.Emoji[i]
	}
	return allEmojis, nil
}

func getEmojis(page int) ([]byte, error) {
	vals := url.Values{}
	vals.Set("token", ownerUserOauthToken)
	vals.Set("page", strconv.Itoa(page))
	vals.Set("count", strconv.Itoa(pageSize))
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
	responseParsed.Emoji = make([]*emoji, 0, pageSize)
	err := json.Unmarshal(response, &responseParsed)
	if err != nil {
		return nil, err
	}
	return &responseParsed, nil
}
