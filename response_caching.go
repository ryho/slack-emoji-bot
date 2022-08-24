package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/ryho/slack-emoji-bot/util"
)

const (
	snapshotDir = "/Documents/emojiSnapshots/"
	imagesDir   = snapshotDir + "images/"
)

func ensureDirExists(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.Mkdir(path, 0777)
	}
	return nil
}

func cacheEmojiResponse(commandResponse *SlackEmojiResponseMessage) error {
	userDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	err = ensureDirExists(userDir + snapshotDir)
	if err != nil {
		return err
	}
	responseBytes, err := json.Marshal(commandResponse)
	if err != nil {
		return err
	}
	fileName := userDir + snapshotDir + time.Now().String() + ".json"
	return ioutil.WriteFile(fileName, responseBytes, 0644)
}

func cacheEmojiImages(response *SlackEmojiResponseMessage) error {
	if cacheImages {
		userDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		err = ensureDirExists(userDir + imagesDir)
		if err != nil {
			return err
		}
		// Download images for all emojis
		for _, emoji := range response.Emoji {
			if strings.HasPrefix(emoji.Url, "data:") {
				// Handle base64 images
				i := strings.Index(emoji.Url, ";")
				ext := emoji.Url[len("data:image/"):i]
				imagePath := userDir + imagesDir + emoji.Name + "." + ext
				_, err := os.Stat(imagePath)
				if os.IsNotExist(err) {
					i = strings.Index(emoji.Url, ",")
					dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(emoji.Url[i+1:]))
					output, err := ioutil.ReadAll(dec)
					if err != nil {
						return err
					}
					err = ioutil.WriteFile(imagePath, output, 0644)
					if err != nil {
						return err
					}
				}
			} else {
				// Handle URL images
				imagePath := userDir + imagesDir + emoji.Name + path.Ext(emoji.Url)
				_, err := os.Stat(imagePath)
				if os.IsNotExist(err) {
					resp, err := http.Get(emoji.Url)
					if err != nil {
						return err
					}
					file, err := os.Create(imagePath)
					if err != nil {
						return err
					}
					_, err = io.Copy(file, resp.Body)
					if err != nil {
						return err
					}
					err = file.Close()
					if err != nil {
						return err
					}
				} else if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func detectDeletedEmojis(response *SlackEmojiResponseMessage) error {
	var message string
	lastResponseBytes, err := readLastEmojiDump(1)
	if err != nil {
		return err
	}

	if lastResponseBytes != nil {
		allCurrentEmojis := make(util.StringSet)
		for _, emoji := range response.Emoji {
			allCurrentEmojis[emoji.Name] = util.SetEntry{}
		}
		lastResponse, err := parseEmojiResponse(lastResponseBytes)
		if err != nil {
			return err
		}
		message += "\nDeleted Emojis:\n\n"
		var missingEmojis []*emoji
		var peopleIds []string
		for _, emoji := range lastResponse.Emoji {
			if _, ok := allCurrentEmojis[emoji.Name]; !ok {
				missingEmojis = append(missingEmojis, emoji)
				peopleIds = append(peopleIds, emoji.UserId)
			}
		}
		if len(peopleIds) > 0 {
			userMap, err := getUsers(peopleIds)
			if err != nil {
				return err
			}
			for _, emoji := range missingEmojis {
				user := userMap[emoji.UserId]
				message += fmt.Sprintf("%s (@%s) %v %s \n", emoji.Name, user.Name, time.Unix(int64(emoji.Created), 0), emoji.Url)
			}
		}
		message += "\n"
	}
	_, err = printMessage(MSG_TYPE__REVIEW_ONLY, message)
	return err
}

func readLastEmojiDump(offset int) ([]byte, error) {
	if offset < 0 {
		return nil, fmt.Errorf("negative offset not allowed. Offset was %d", offset)
	}
	dirname, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	files, err := ioutil.ReadDir(dirname + snapshotDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	fileNames := make([]string, 0, len(files))
	for _, file := range files {
		// Skip directories and hidden files
		if !file.IsDir() && file.Name()[0] != '.' {
			fileNames = append(fileNames, file.Name())
		}
	}
	if len(fileNames) <= offset {
		return nil, nil
	}
	sort.Strings(fileNames)
	selectedName := fileNames[len(fileNames)-1-offset]
	fileContents, err := ioutil.ReadFile(dirname + snapshotDir + selectedName)
	if err != nil {
		return nil, err
	}
	return fileContents, nil
}
