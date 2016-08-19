package bot

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/nlopes/slack"
	"github.com/uber-go/zap"
)

var isImage = regexp.MustCompile("\\.(jpe?g|gif|png)$")
var CdnPath = ""
var CdnPrefix = ""

func fileBytes(f *slack.File) ([]byte, error) {
	req, _ := http.NewRequest("GET", f.URLPrivate, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		return nil, fmt.Errorf(rsp.Status)
	}
	return ioutil.ReadAll(rsp.Body)
}

func handleChannelUpload(m *slack.MessageEvent) bool {
	if CdnPath == "" || CdnPrefix == "" {
		return false
	}
	if !m.Msg.Upload {
		return false
	}
	logger.Info("File upload detected", zap.String("username", m.Username), zap.String("filename", m.File.Name))
	if buf, err := fileBytes(m.Msg.File); err != nil {
		logger.Error(
			"error downloading file",
			zap.Error(err),
			zap.String("username", m.Username),
			zap.String("filename", m.File.Name))
	} else {
		path := fmt.Sprintf("%s/%s", CdnPath, time.Now().Format("2006/01/02/15"))
		if err := os.MkdirAll(path, 0755); err != nil {
			logger.Error(
				"error making cdn path",
				zap.String("path", path),
				zap.String("username", m.Username),
				zap.String("filename", m.File.Name))
			return false
		}
		part := &url.URL{Path: m.Msg.File.Name}
		urlPath := fmt.Sprintf("%s/%s-%s", path, m.Msg.File.ID, part.String())
		path = fmt.Sprintf("%s/%s-%s", path, m.Msg.File.ID, m.Msg.File.Name)
		if fp, err := os.Create(path); err != nil {
			logger.Error(
				"error creating cdn file",
				zap.String("path", path),
				zap.String("username", m.Username),
				zap.String("filename", m.File.Name))
			return false
		} else {
			if _, err := fp.Write(buf); err != nil {
				fp.Close()
				logger.Error(
					"error writing to cdn file",
					zap.String("path", path),
					zap.String("username", m.Username),
					zap.String("filename", m.File.Name))
				return false
			}
			fp.Close()
			fileURL := CdnPrefix + urlPath[len(CdnPath):]
			rtm.DeleteFile(m.Msg.File.ID)
			if isImage.MatchString(strings.ToLower(m.Msg.File.Name)) {
				for i := 0; i < 5; i++ {
					_, _, err := rtm.PostMessage(
						m.Channel,
						"",
						slack.PostMessageParameters{
							Text:        "",
							AsUser:      true,
							UnfurlLinks: true,
							UnfurlMedia: true,
							IconEmoji:   ":paperclip:",
							Attachments: []slack.Attachment{
								slack.Attachment{
									Title:     fmt.Sprintf("%s uploaded %s", m.Msg.Username, m.Msg.File.Title),
									TitleLink: fileURL,
									ImageURL:  fileURL,
								},
							},
						})
					if err != nil {
						logger.Error(
							"Failed postting cdn link back to slack",
							zap.String("username", m.Username),
							zap.String("filename", m.File.Name),
							zap.String("url", fileURL))
					} else {
						break
					}
					time.Sleep(time.Second * time.Duration(i))
				}
			} else {
				rtm.SendMessage(&slack.OutgoingMessage{
					ID:      int(time.Now().UnixNano()),
					Channel: m.Channel,
					Text:    fmt.Sprintf("%s uploaded the file *%s*\n%s", m.Msg.Username, m.Msg.File.Title, fileURL),
					Type:    "message",
				})
			}
			logger.Info("saved CDN file", zap.String("url", fileURL), zap.Int("size", len(buf)))
			return true
		}

	}
	return false
}

func handleDMUpload(m *slack.MessageEvent) bool {
	if CdnPath == "" || CdnPrefix == "" {
		return false
	}
	if !m.Msg.Upload {
		return false
	}
	if buf, err := fileBytes(m.Msg.File); err != nil {
		logger.Info("error downloading file", zap.Error(err))
	} else {
		path := fmt.Sprintf("%s/%s", CdnPath, time.Now().Format("2006/01/02/15"))
		if err := os.MkdirAll(path, 0755); err != nil {
			logger.Error("error making cdn path", zap.String("path", path))
			return false
		}
		part := &url.URL{Path: m.Msg.File.Name}
		urlPath := fmt.Sprintf("%s/%s-%s", path, m.Msg.File.ID, part.String())
		path = fmt.Sprintf("%s/%s-%s", path, m.Msg.File.ID, m.Msg.File.Name)
		if fp, err := os.Create(path); err != nil {
			logger.Error("error creating cdn file", zap.String("path", path))
			return false
		} else {
			if _, err := fp.Write(buf); err != nil {
				fp.Close()
				logger.Error("error writing to cdn file", zap.String("path", path))
				return false
			}
			fp.Close()
			fileURL := CdnPrefix + urlPath[len(CdnPath):]
			rtm.DeleteFile(m.Msg.File.ID)
			rtm.SendMessage(&slack.OutgoingMessage{
				ID:      int(time.Now().UnixNano()),
				Channel: m.Channel,
				Text:    fmt.Sprintf("Thanks for sending me the file instead of uploading it to a channel or group. You can paste the following link anywhere you want to show the file to others! ```%s```", fileURL),
				Type:    "message",
			})
			logger.Info("saved CDN file", zap.String("url", fileURL), zap.Int("size", len(buf)))
		}

	}
	return true
}
