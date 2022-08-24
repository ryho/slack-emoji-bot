package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/slack-go/slack"
)

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

func createNameString(peopleArray []string) string {
	if len(peopleArray) == 0 {
		return ""
	} else if len(peopleArray) == 1 {
		return "Thanks to " + peopleArray[0] + "."
	}
	sort.Strings(peopleArray)
	return "Thanks to " + strings.Join(peopleArray[:len(peopleArray)-1], ", ") + ", and " + peopleArray[len(peopleArray)-1] + "."
}

func topAndNewUploaders(response *SlackEmojiResponseMessage) error {
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
	err = printTopPeople(topAllTimeMessage, topSecondMessage, people, maxPeopleForTopUploaders, !sendTopUploadersAllTime)
	if err != nil {
		return err
	}

	// Find people who uploaded for the first time.
	newPeopleThisWeek := map[string]*stringCount{}
	for _, uploadThisWeek := range response.peopleThisWeek {
		if uploadsAllTime, ok := people[uploadThisWeek.id]; !ok || uploadThisWeek.count == uploadsAllTime.count {
			newPeopleThisWeek[uploadThisWeek.id] = uploadThisWeek
		}
	}
	newUploadersMessage := fmt.Sprintf(newUploadersMessage, len(newPeopleThisWeek))
	err = printTopPeople(newUploadersMessage, newUploadersSecondMessage, newPeopleThisWeek, maxPeopleForTopUploaders, false)
	if err != nil {
		return err
	}

	return nil
}

func printTopPeople(firstMessage, secondMessage string, people map[string]*stringCount, maxPeople int, printOnly bool) error {
	var peopleCountArray []*stringCount
	for _, count := range people {
		peopleCountArray = append(peopleCountArray, count)
	}
	sort.Sort(ByCount(peopleCountArray))
	firstMessage += "\n"
	secondMessage += "\n"
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
				firstMessage += printer.Sprintf("%d. %s (%s) :%s: %d\n", i+1, user.RealName, user.Name, emojis[i], reactions[i])
			} else {
				secondMessage += printer.Sprintf("%d. %s (%s) :%s: %d\n", i+1, user.RealName, user.Name, emojis[i], reactions[i])
			}
		} else {
			if i < TopPeopleToPrint {
				if runMode == MODE__PRINT_EVERYTHING || runMode == MODE__DM_FOR_REVIEW {
					firstMessage += printer.Sprintf("%d. %s (@%s) :%s: %d\n", i+1, user.RealName, user.Name, emojis[i], reactions[i])
				} else {
					// Since this will be sent to the API, use the API format.
					firstMessage += printer.Sprintf("%d. %s (<@%s>) :%s: %d\n", i+1, user.RealName, user.ID, emojis[i], reactions[i])
				}
			} else {
				if runMode == MODE__PRINT_EVERYTHING || runMode == MODE__DM_FOR_REVIEW {
					secondMessage += printer.Sprintf("%d. %s (@%s) :%s: %d\n", i+1, user.RealName, user.Name, emojis[i], reactions[i])
				} else {
					// Since this will be sent to the API, use the API format.
					secondMessage += printer.Sprintf("%d. %s (<@%s>) :%s: %d\n", i+1, user.RealName, user.ID, emojis[i], reactions[i])
				}
			}
		}
	}
	threadId, err := printMessage(MSG_TYPE__SEND_AND_REVIEW, firstMessage)
	if err != nil {
		return err
	}
	secondMessage += "\n" + fmt.Sprintf(muteMessage, ownerLDAP)
	secondMessage += "\n" + fmt.Sprintf(skipMessage, ownerLDAP)

	_, err = printMessageWithThreadId(MSG_TYPE__SEND_AND_REVIEW, secondMessage, threadId)
	if err != nil {
		return err
	}
	return nil
}
