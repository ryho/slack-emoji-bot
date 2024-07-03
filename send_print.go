package main

import (
	"fmt"
	"time"

	"github.com/slack-go/slack"
)

type MessageType int

const (
	MSG_TYPE__SEND MessageType = iota
	MSG_TYPE__REVIEW_ONLY
	MSG_TYPE__SEND_AND_REVIEW
	MSG_TYPE__DM_ONLY
	MSG_TYPE__PRINT_ONLY

	slackRateLimitFormat = `slack rate limit exceeded, retry after %ds`
)

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
	if err != nil {
		// If we got rate limited, wait the amount of time that it recommends.
		var sleepTime time.Duration
		_, err2 := fmt.Sscanf(err.Error(), slackRateLimitFormat, &sleepTime)
		if err2 != nil || sleepTime == 0 {
			return "", err
		} else {
			fmt.Printf("Sleeping for %d seconds per rate limit message\n", sleepTime)
			time.Sleep(sleepTime * time.Second)
			_, msgId, err = slackApi.PostMessage(dest, options...)
		}
	}
	return msgId, err
}

func GetConversationsWithBackoff(channelsParams *slack.GetConversationsParameters) (channels []slack.Channel, nextCursor string, err error) {
	channels, nextCursor, err = slackApi.GetConversations(channelsParams)
	if err != nil {
		// If we got rate limited, wait the amount of time that it recommends.
		var sleepTime time.Duration
		_, err2 := fmt.Sscanf(err.Error(), slackRateLimitFormat, &sleepTime)
		if err2 != nil || sleepTime == 0 {
			return nil, "", err
		} else {
			fmt.Printf("Sleeping for %d seconds per rate limit message\n", sleepTime)
			time.Sleep(sleepTime * time.Second)
			channels, nextCursor, err = slackApi.GetConversations(channelsParams)
		}
	}

	return channels, nextCursor, err
}

func GetConversationHistoryWithBackoff(channelsParams *slack.GetConversationHistoryParameters) (resp *slack.GetConversationHistoryResponse, err error) {
	resp, err = slackApi.GetConversationHistory(channelsParams)
	if err != nil {
		// If we got rate limited, wait the amount of time that it recommends.
		var sleepTime time.Duration
		_, err2 := fmt.Sscanf(err.Error(), slackRateLimitFormat, &sleepTime)
		if err2 != nil || sleepTime == 0 {
			return nil, err
		} else {
			fmt.Printf("Sleeping for %d seconds per rate limit message\n", sleepTime)
			time.Sleep(sleepTime * time.Second)
			resp, err = slackApi.GetConversationHistory(channelsParams)
		}
	}

	return resp, err
}
