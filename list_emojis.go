package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	emojiListUrl = "https://square.slack.com/api/emoji.adminList"
)

var (
	pageSize = 10000
)

func init() {
	if fastMode {
		pageSize = 1000
	}
}

type SlackEmojiResponseMessage struct {
	Ok                    bool           `json:"ok"`
	Error                 string         `json:"error"`
	Emoji                 []*emoji       `json:"emoji"`
	CustomEmojiTotalCount int64          `json:"custom_emoji_total_count"`
	Paging                PagingResponse `json:"paging"`
	emojiMap              map[string]*emoji
	peopleThisWeek        map[string]*stringCount
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
	for page := 1; currentPage == nil || page <= currentPage.Paging.Pages; page++ {
		fmt.Printf("Getting page %v\n", page)
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

func getEmojisBackTo(lastEmoji string) (*SlackEmojiResponseMessage, error) {
	var allEmojis, currentPage *SlackEmojiResponseMessage
	for page := 1; currentPage == nil || page <= currentPage.Paging.Pages; page++ {
		fmt.Printf("Getting page %v\n", page)
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
		stop := false
		for _, emoji := range currentPage.Emoji {
			if emoji.Name == lastEmoji {
				stop = true
			}
		}
		if stop {
			break
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

	req, err := http.NewRequest("POST", emojiListUrl, strings.NewReader(vals.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("cookie", ownerUserCookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bodyBytes, nil
}

func parseEmojiResponse(response []byte) (responseParsed *SlackEmojiResponseMessage, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Error parsing emoji response: %v", string(response))
		}
	}()
	//fmt.Printf("Parsing response: %v\n", string(response))
	responseParsed = &SlackEmojiResponseMessage{}
	responseParsed.Emoji = make([]*emoji, 0, pageSize)
	err = json.Unmarshal(response, responseParsed)
	if err != nil {
		return nil, err
	}
	if !responseParsed.Ok {
		return nil, fmt.Errorf("recieved error from Slack: %v", responseParsed.Error)
	}
	return responseParsed, nil
}
