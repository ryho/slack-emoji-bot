package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// START of things you should edit
// See config.go for more things to edit

// Caching all emoji images can take a lot of time and storage, so you
// may want to leave it off. It takes 547MB for my company's Slack.
// The only use of cashing all images is to see what deleted emojis were.
// After all emojis have been downloaded the first time, this will only download new emojis.
// You can also download the image for a deleted emoji if it has not been deleted for very long.
const cacheImages = true

// This controls if the JSON blob of all current emojis is cached. This is only used for detecting
// deleted emojis.
const cacheEmojiDumps = true
const runMode Mode = MODE__PRINT_EVERYTHING

// Top uploaders of all time is noisy.
// I only send at the end of the year, if someone has recently moved up a lot, etc.
const sendTopUploadersAllTime = false

// The channel to post these messages in when in FULL_SEND mode.
const emojiChannel = "#emojis"

// END of things you should edit
// See config.go for more things to edit

type Mode int

const (
	MODE__PRINT_EVERYTHING Mode = iota
	MODE__DM_FOR_REVIEW
	MODE__DM_FOR_TESTING
	MODE__FULL_SEND
)

var slackApi *slack.Client

func main() {
	slackApi = slack.New(botOauthToken)
	allEmojis, err := getAllEmojis()
	if err != nil {
		panic(err)
	}

	err = printTopEmojis(lastWeekReactions, allEmojis)
	if err != nil {
		panic(err)
	}

	// cacheEmojiImages and detectDeletedEmojis should be called before removeSkippedEmojis
	err = cacheEmojiImages(allEmojis)
	if err != nil {
		panic(err)
	}

	err = detectDeletedEmojis(allEmojis)
	if err != nil {
		panic(err)
	}

	removeSkippedEmojis(allEmojis)

	// mostRecentEmojis, topUploaders, and longestEmojis should be called after removeSkippedEmojis
	err = mostRecentEmojis(allEmojis)
	if err != nil {
		panic(err)
	}

	err = topUploaders(allEmojis)
	if err != nil {
		panic(err)
	}

	err = longestEmojis(allEmojis)
	if err != nil {
		panic(err)
	}
}

const (
	maxEmojisPerMessage       = 22
	maxPeopleForTopUploaders  = 100
	maxEmojisForLongestEmojis = 100
	maxCharactersPerMessage   = 10000
	TopPeopleToPrint          = 5

	lastWeek           = "Congratulations to the top emojis from last week (sorted by emoji reactions):\n"
	introMessage       = "Here are all the new emojis! There are %v new emojis from %v people."
	lastMessage        = "React here with the best new emojis!"
	topAllTimeMessage  = "Top Emoji Uploaders of All Time:"
	topThisWeekMessage = "Top Emoji Uploaders This Week:"
	muteMessage        = "If you do not want to be pinged by this list, message @%s to request that you be added to the mute list so the script prints your name without the @ sign.\n"
	skipMessage        = "If you want to be excluded from the list all together, you can ask @%s to add you to the skip list.\n"

	snapshotDir = "/Documents/emojiSnapshots/"
	imagesDir   = snapshotDir + "images/"

	emojiListUrl = "https://square.slack.com/api/emoji.adminList"
)

type MessageType int

const (
	MSG_TYPE__SEND MessageType = iota
	MSG_TYPE__REVIEW_ONLY
	MSG_TYPE__SEND_AND_REVIEW
	MSG_TYPE__DM_ONLY
	MSG_TYPE__PRINT_ONLY
)

// This takes in the copied text of a message from Slack, and prints the top Emoji reactions.
func printTopEmojis(reactions string, allEmojis *SlackEmojiResponseMessage) error {
	lines := strings.Split(reactions, "\n")
	var emojis []*stringCount
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, ":") && strings.HasSuffix(line, ":") {
			if i+1 >= len(lines) {
				continue
			}
			count, err := strconv.Atoi(lines[i+1])
			if err != nil {
				continue
			}
			emojis = append(emojis, &stringCount{name: line, count: count})
		}
	}
	sort.Sort(ByCount(emojis))

	printedCount := 0
	previousCount := math.MaxInt64

	maxPrintCount := 10
	minReaction := 3
	var creators []string
	var counts []int
	var printedEmojis []string
	for _, emoji := range emojis {
		// Stop if we have printed enough emojis, however, always print all emojis with the same reaction
		// count even if we go over the limit.
		if emoji.count != previousCount && printedCount >= maxPrintCount {
			break
		}
		// Stop if the reaction count is too low, even if we have not hit the limit.
		if emoji.count < minReaction {
			break
		}
		emojisObj := allEmojis.emojiMap[emoji.name[1:len(emoji.name)-1]]
		creators = append(creators, emojisObj.UserId)
		counts = append(counts, emoji.count)
		printedEmojis = append(printedEmojis, emoji.name)
		previousCount = emoji.count
		printedCount++
	}

	return printTopCreators(lastWeek, creators, counts, printedEmojis)
}

var (
	printer = message.NewPrinter(language.English)
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

type stringCount struct {
	count int
	name  string
	id    string
}

func parseEmojiResponse(response []byte) (*SlackEmojiResponseMessage, error) {
	var responseParsed SlackEmojiResponseMessage
	err := json.Unmarshal(response, &responseParsed)
	if err != nil {
		return nil, err
	}
	return &responseParsed, nil
}

func ensureDirExists(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.Mkdir(path, 0777)
	}
	return nil
}

func getAllEmojis() (*SlackEmojiResponseMessage, error) {
	commandResponse, err := getEmojis()
	if err != nil {
		return nil, err
	}
	userDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	if cacheEmojiDumps {
		err = ensureDirExists(userDir + snapshotDir)
		if err != nil {
			return nil, err
		}
		fileName := userDir + snapshotDir + time.Now().String() + ".json"
		err = ioutil.WriteFile(fileName, commandResponse, 0644)
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

func getUsers(userIds []string) (map[string]*slack.User, error) {
	users := map[string]*slack.User{}
	var endIndex int
	// This endpoint only supports 100 users per request, so we need to request them in batches.
	for startIndex := 0; startIndex < len(userIds); startIndex = endIndex {
		endIndex = minInt(startIndex+30, len(userIds))
		batchUserIds := userIds[startIndex:endIndex]
		usersResults, err := slackApi.GetUsersInfo(batchUserIds...)
		if err != nil {
			return nil, err
		}
		for i, user := range *usersResults {
			users[user.ID] = &(*usersResults)[i]
		}
	}
	return users, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func detectDeletedEmojis(response *SlackEmojiResponseMessage) error {
	var message string
	lastResponseBytes, err := readLastEmojiDump(1)
	if err != nil {
		return err
	}

	if lastResponseBytes != nil {
		allCurrentEmojis := make(map[string]struct{})
		for _, emoji := range response.Emoji {
			allCurrentEmojis[emoji.Name] = struct{}{}
		}
		lastResponse, err := parseEmojiResponse(lastResponseBytes)
		if err != nil {
			return err
		}
		message += "\nDeleted Emojis:\n\n"
		var missingEmojis []emoji
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
	_, err = printMessage(MSG_TYPE__PRINT_ONLY, message)
	return err
}

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

func printMessage(level MessageType, text string) (string, error) {
	return printMessageWithThreadId(level, text, "")
}
func printMessageWithThreadId(level MessageType, text string, threadId string) (string, error) {
	switch level {
	case MSG_TYPE__SEND:
		if runMode == MODE__PRINT_EVERYTHING {
			fmt.Print("\n\n" + text + "\n\n")
			return "", nil
		} else if runMode == MODE__FULL_SEND {
			return sendMessage(emojiChannel, text, threadId)
		} else if runMode == MODE__DM_FOR_REVIEW {
			fmt.Print("\n\n" + text + "\n\n")
			return "", nil
		} else if runMode == MODE__DM_FOR_TESTING {
			return sendMessage(ownerUserId, text, threadId)
		}
	case MSG_TYPE__REVIEW_ONLY:
		if runMode == MODE__DM_FOR_REVIEW {
			var firstTS string
			for _, id := range append(additionalReviewerIds, ownerUserId) {
				ts, err := sendMessage(id, text, threadId)
				if err != nil {
					return "", err
				}
				threadId = ""
				if firstTS != "" {
					firstTS = ts
				}
			}
			return firstTS, nil
		} else {
			return "", nil
		}
	case MSG_TYPE__SEND_AND_REVIEW:
		if runMode == MODE__PRINT_EVERYTHING {
			fmt.Print("\n\n" + text + "\n\n")
			return "", nil
		} else if runMode == MODE__FULL_SEND {
			return sendMessage(emojiChannel, text, threadId)
		} else if runMode == MODE__DM_FOR_REVIEW {
			var firstTS string
			for _, id := range append(additionalReviewerIds, ownerUserId) {
				ts, err := sendMessage(id, text, threadId)
				if err != nil {
					return "", err
				}
				threadId = ""
				if firstTS != "" {
					firstTS = ts
				}
			}
			return firstTS, nil
		} else if runMode == MODE__DM_FOR_TESTING {
			return sendMessage(ownerUserId, text, threadId)
		}
	case MSG_TYPE__DM_ONLY:
		if runMode == MODE__PRINT_EVERYTHING {
			fmt.Print("\n\n" + text + "\n\n")
			return "", nil
		} else if runMode == MODE__FULL_SEND {
			fmt.Print("\n\n" + text + "\n\n")
			return "", nil
		} else if runMode == MODE__DM_FOR_REVIEW {
			var firstTS string
			for _, id := range append(additionalReviewerIds, ownerUserId) {
				ts, err := sendMessage(id, text, threadId)
				if err != nil {
					return "", err
				}
				threadId = ""
				if firstTS != "" {
					firstTS = ts
				}
			}
			return firstTS, nil
		} else if runMode == MODE__DM_FOR_TESTING {
			fmt.Print("\n\n" + text + "\n\n")
			return "", nil
		}
	case MSG_TYPE__PRINT_ONLY:
		fmt.Print("\n\n" + text + "\n\n")
	default:
		fmt.Print("\n\n" + text + "\n\n")
	}
	return "", nil
}

func sendMessage(dest, text, threadId string) (string, error) {
	var options = []slack.MsgOption{slack.MsgOptionText(text, false)}
	if threadId != "" {
		options = append(options, slack.MsgOptionTS(threadId))
	}
	_, msgId, err := slackApi.PostMessage(dest, options...)
	return msgId, err
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

func mostRecentEmojis(response *SlackEmojiResponseMessage) error {
	var allNewEmojis []string
	people := map[string]*stringCount{}
	var foundLastEmoji bool
	lastNewEmojiSanitized := strings.ReplaceAll(lastNewEmoji, ":", "")

	sort.Sort(EmojiUploadDateSort(response.Emoji))

	for _, emoji := range response.Emoji {
		if emoji.Name == lastNewEmojiSanitized {
			foundLastEmoji = true
			break
		}
		count, ok := people[emoji.UserId]
		if !ok {
			people[emoji.UserId] = &stringCount{
				name:  emoji.UserDisplayName,
				id:    emoji.UserId,
				count: 1,
			}
		} else {
			count.count++
		}
		allNewEmojis = append(allNewEmojis, emoji.Name)
	}
	if !foundLastEmoji {
		fmt.Printf("Did not find the last emoji %v. This is probably a problem.\n", lastNewEmoji)
	}

	slackMessages := make([]string, 0, len(response.Emoji)/maxEmojisPerMessage+1)
	i := 0
	j := 0
	auditMessage := []string{""}
	for z := len(allNewEmojis) - 1; z >= 0; z-- {
		emojiName := allNewEmojis[z]
		if emojiName == lastNewEmojiSanitized {
			foundLastEmoji = true
			break
		}
		newPart := ":" + emojiName + ": " + emojiName + "\n"
		if len(newPart)+len(auditMessage[len(auditMessage)-1]) > maxCharactersPerMessage {
			auditMessage = append(auditMessage, newPart)
		} else {
			auditMessage[len(auditMessage)-1] += newPart
		}
		if len(slackMessages) > i {
			slackMessages[i] = slackMessages[i] + ":" + emojiName + ":"
		} else {
			slackMessages = append(slackMessages, ":"+emojiName+":")
		}
		if j == maxEmojisPerMessage {
			i++
			j = 0
		} else {
			j++
		}

	}

	_, err := printMessage(MSG_TYPE__SEND, printer.Sprintf(introMessage, len(allNewEmojis), len(people)))
	if err != nil {
		return err
	}

	for _, msg := range slackMessages {
		_, err := printMessage(MSG_TYPE__SEND_AND_REVIEW, msg)
		if err != nil {
			return err
		}
	}
	var peopleNameArray []string
	for _, person := range people {
		peopleNameArray = append(peopleNameArray, person.name)
	}

	threadId, err := printMessage(MSG_TYPE__SEND, lastMessage)
	if err != nil {
		return err
	}
	// Side by side message
	for _, part := range auditMessage {
		_, err := printMessageWithThreadId(MSG_TYPE__SEND, part, threadId)
		if err != nil {
			return err
		}
	}

	// List everyone's names
	_, err = printMessageWithThreadId(MSG_TYPE__SEND, createNameString(peopleNameArray), threadId)
	if err != nil {
		return err
	}

	return printTopPeople(topThisWeekMessage, people, math.MaxInt64, false)
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

func createNameString(peopleArray []string) string {
	if len(peopleArray) == 0 {
		return ""
	} else if len(peopleArray) == 1 {
		return "Thanks to " + peopleArray[0] + "."
	}
	sort.Strings(peopleArray)
	return "Thanks to " + strings.Join(peopleArray[:len(peopleArray)-1], ", ") + ", and " + peopleArray[len(peopleArray)-1] + "."
}

func topUploaders(response *SlackEmojiResponseMessage) error {
	people := map[string]*stringCount{}
	for _, emoji := range response.Emoji {
		count, ok := people[emoji.UserId]
		if !ok {
			people[emoji.UserId] = &stringCount{
				name:  emoji.UserDisplayName,
				id:    emoji.UserId,
				count: 1,
			}
		} else {
			count.count++
		}
	}
	_, err := printer.Printf("%d people have uploaded %d emojis\n", len(people), len(response.Emoji))
	if err != nil {
		return err
	}
	return printTopPeople(topAllTimeMessage, people, maxPeopleForTopUploaders, !sendTopUploadersAllTime)
}

func longestEmojis(response *SlackEmojiResponseMessage) error {
	sort.Sort(StringLengthSort(response.Emoji))
	message := "Longest Emoji Names:\n"
	for i := 0; i < maxEmojisForLongestEmojis && i < len(response.Emoji); i++ {
		message += printer.Sprintf("%d. :%s: %s (%d)\n", i+1, response.Emoji[i].Name, response.Emoji[i].Name, len(response.Emoji[i].Name))
	}
	_, err := printMessage(MSG_TYPE__PRINT_ONLY, message)
	return err
}

func printTopPeople(message string, people map[string]*stringCount, maxPeople int, printOnly bool) error {
	var peopleCountArray []*stringCount
	for _, count := range people {
		peopleCountArray = append(peopleCountArray, count)
	}
	sort.Sort(ByCount(peopleCountArray))
	var firstMessage, secondMessage string
	firstMessage = message + "\n"
	secondMessage = "More Top Emoji Uploaders This Week!\n"
	var peopleIds []string
	for i := 0; i < maxPeople && i < len(peopleCountArray); i++ {
		peopleIds = append(peopleIds, peopleCountArray[i].id)
	}
	userMap, err := getUsers(peopleIds)
	if err != nil {
		return err
	}
	var skipCorrection int
	for i := 0; i < maxPeople && i < len(peopleCountArray); i++ {
		user, ok := userMap[peopleCountArray[i].id]
		if !ok {
			return fmt.Errorf("could not find user %v %v", peopleCountArray[i].id, peopleCountArray[i].name)
		}
		if _, ok := skipLDAPs[user.Name]; ok {
			// This skips the user so they do not show up at all.
			skipCorrection++
			continue
		}
		if _, ok := muteLDAPs[user.Name]; ok {
			// This prints the LDAP with no @ sign, so they will not be pinged.
			if i < TopPeopleToPrint {
				firstMessage += printer.Sprintf("%d. %s (%s) %d\n", i+1-skipCorrection, peopleCountArray[i].name, user.Name, peopleCountArray[i].count)
			} else {
				secondMessage += printer.Sprintf("%d. %s (%s) %d\n", i+1-skipCorrection, peopleCountArray[i].name, user.Name, peopleCountArray[i].count)
			}
		} else {
			if i < TopPeopleToPrint {
				if printOnly || runMode == MODE__PRINT_EVERYTHING || runMode == MODE__DM_FOR_REVIEW {
					firstMessage += printer.Sprintf("%d. %s (@%s) %d\n", i+1-skipCorrection, peopleCountArray[i].name, user.Name, peopleCountArray[i].count)
				} else {
					// Since this will be sent to the API, use the API format.
					firstMessage += printer.Sprintf("%d. %s (<@%s>) %d\n", i+1-skipCorrection, peopleCountArray[i].name, user.ID, peopleCountArray[i].count)
				}
			} else {
				if printOnly || runMode == MODE__PRINT_EVERYTHING || runMode == MODE__DM_FOR_REVIEW {
					secondMessage += printer.Sprintf("%d. %s (@%s) %d\n", i+1-skipCorrection, peopleCountArray[i].name, user.Name, peopleCountArray[i].count)
				} else {
					// Since this will be sent to the API, use the API format.
					secondMessage += printer.Sprintf("%d. %s (<@%s>) %d\n", i+1-skipCorrection, peopleCountArray[i].name, user.ID, peopleCountArray[i].count)
				}
			}
		}
		if err != nil {
			return err
		}
	}
	var threadId string
	if printOnly {
		_, err = printMessage(MSG_TYPE__PRINT_ONLY, firstMessage)
	} else {
		threadId, err = printMessage(MSG_TYPE__SEND, firstMessage)
	}
	if err != nil {
		return err
	}
	secondMessage += fmt.Sprintf(muteMessage, ownerLDAP)
	secondMessage += fmt.Sprintf(skipMessage, ownerLDAP)

	if printOnly {
		_, err = printMessage(MSG_TYPE__PRINT_ONLY, secondMessage)
	} else {
		_, err = printMessageWithThreadId(MSG_TYPE__SEND, secondMessage, threadId)
	}
	return err
}

func printTopCreators(message string, peopleIds []string, reactions []int, emojis []string) error {
	var firstMessage, secondMessage string
	firstMessage = message
	secondMessage = "More Top Uploaders\n"
	userMap, err := getUsers(peopleIds)
	if err != nil {
		return err
	}
	for i, peopleId := range peopleIds {
		user, ok := userMap[peopleId]
		if !ok {
			return fmt.Errorf("could not find user %v", peopleId)
		}
		if _, ok := skipLDAPs[user.Name]; ok {
			continue
		}
		if _, ok := muteLDAPs[user.Name]; ok {
			// This prints the LDAP with no @ sign, so they will not be pinged.
			if i < TopPeopleToPrint {
				firstMessage += printer.Sprintf("%d. %s (%s) %s %d\n", i+1, user.RealName, user.Name, emojis[i], reactions[i])
			} else {
				secondMessage += printer.Sprintf("%d. %s (%s) %s %d\n", i+1, user.RealName, user.Name, emojis[i], reactions[i])
			}
		} else {
			if i < TopPeopleToPrint {
				if runMode == MODE__PRINT_EVERYTHING || runMode == MODE__DM_FOR_REVIEW {
					firstMessage += printer.Sprintf("%d. %s (@%s) %s %d\n", i+1, user.RealName, user.Name, emojis[i], reactions[i])
				} else {
					// Since this will be sent to the API, use the API format.
					firstMessage += printer.Sprintf("%d. %s (<@%s>) %s %d\n", i+1, user.RealName, user.ID, emojis[i], reactions[i])
				}
			} else {
				secondMessage += printer.Sprintf("%d. %s (@%s) %s %d\n", i+1, user.RealName, user.Name, emojis[i], reactions[i])
			}
		}
	}
	_, err = printMessage(MSG_TYPE__SEND, firstMessage)
	if err != nil {
		return err
	}
	secondMessage += "\n" + fmt.Sprintf(muteMessage, ownerLDAP)
	secondMessage += "\n" + fmt.Sprintf(skipMessage, ownerLDAP)

	_, err = printMessage(MSG_TYPE__PRINT_ONLY, secondMessage)
	if err != nil {
		return err
	}
	return nil
}

type StringLengthSort []emoji

func (p StringLengthSort) Len() int           { return len(p) }
func (p StringLengthSort) Less(i, j int) bool { return len(p[i].Name) > len(p[j].Name) }
func (p StringLengthSort) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type EmojiUploadDateSort []emoji

func (p EmojiUploadDateSort) Len() int           { return len(p) }
func (p EmojiUploadDateSort) Less(i, j int) bool { return p[i].Created > p[j].Created }
func (p EmojiUploadDateSort) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type ByCount []*stringCount

func (a ByCount) Len() int { return len(a) }
func (a ByCount) Less(i, j int) bool {
	if a[i].count == a[j].count {
		return a[i].name < a[j].name
	}
	return a[i].count > a[j].count
}
func (a ByCount) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
