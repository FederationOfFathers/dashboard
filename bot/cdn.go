package bot

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/nlopes/slack"
	"go.uber.org/zap"
)

var isImage = regexp.MustCompile("\\.(jpe?g|gif|png)$")
var CdnPath = ""
var CdnPrefix = ""

func fileBytesQuietly(f *slack.File) ([]byte, error) {
	req, _ := http.NewRequest(
		"GET",
		f.URLPrivate,
		nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		buf, _ := httputil.DumpRequest(req, true)
		bufRsp, _ := httputil.DumpResponse(rsp, false)
		Logger.Info("Debugging File Request",
			zap.String("method", "old"),
			zap.ByteString("request", buf),
			zap.ByteString("response", bufRsp),
		)
		return nil, fmt.Errorf(rsp.Status)
	}
	if strings.Contains(rsp.Header.Get("Content-Type"), "text/html") {
		buf, _ := httputil.DumpRequest(req, true)
		bufRsp, _ := httputil.DumpResponse(rsp, false)
		Logger.Info("Debugging File Request",
			zap.String("method", "old"),
			zap.ByteString("request", buf),
			zap.ByteString("response", bufRsp),
		)
		return nil, fmt.Errorf("Expected non html content type, got %s", rsp.Header.Get("Content-Type"))
	}
	return ioutil.ReadAll(rsp.Body)
}

func fileBytesNoisy(f *slack.File) ([]byte, error) {
	for i := 0; i < 5; i++ {
		if _, _, _, err := rtm.ShareFilePublicURL(f.ID); err == nil {
			break
		} else {
			if i < 4 {
				sleepfor := time.Millisecond * time.Duration(((i+1)*(i+1))*100)
				Logger.Error(
					"Error making file public",
					zap.Error(err),
					zap.String("filename", f.Name),
					zap.String("fileid", f.ID),
					zap.Duration("sleepfor", sleepfor))
			} else {
				Logger.Error(
					"Error making file public: %s/%s: %s",
					zap.Error(err),
					zap.String("filename", f.Name),
					zap.String("fileid", f.ID))
				return nil, fmt.Errorf("Error making file public: %s/%s: %s", f.ID, f.Name, err.Error())
			}
		}

	}
	var pubSecret string
	pubParts := strings.Split(f.PermalinkPublic, "-")
	pubSecret = pubParts[len(pubParts)-1]
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf(
			"%s?pub_secret=%s",
			f.URLPrivate,
			pubSecret,
		),
		nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	buf, _ := httputil.DumpRequest(req, true)
	Logger.Info("Debugging File Request", zap.ByteString("request", buf))
	rsp, err := http.DefaultClient.Do(req)
	bufRsp, _ := httputil.DumpResponse(rsp, false)
	Logger.Info("Debugging File Response", zap.ByteString("response", bufRsp))
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		return nil, fmt.Errorf(rsp.Status)
	}
	if strings.Contains(rsp.Header.Get("Content-Type"), "text/html") {
		return nil, fmt.Errorf("Expected non html content type, got %s", rsp.Header.Get("Content-Type"))
	}
	return ioutil.ReadAll(rsp.Body)
}

func fileBytes(f *slack.File) ([]byte, error) {
	for i := 0; i < 5; i++ {
		if buf, err := fileBytesQuietly(f); err == nil {
			return buf, err
		}
		if i < 4 {
			sleepfor := time.Millisecond * time.Duration(((i+1)*(i+1))*100)
			time.Sleep(sleepfor)
		}

	}
	return fileBytesNoisy(f)
}

func handleChannelUpload(m *slack.MessageEvent) bool {
	if CdnPath == "" || CdnPrefix == "" {
		return false
	}
	if !m.Msg.Upload {
		return false
	}
	file := &m.Files[0]

	// get the username by the ID from the DB
	var user = ":space_invader:"
	profile, err := data.User(m.User)
	if err != nil {
		Logger.Error("Could not get name for user",
			zap.String("userID", m.User),
			zap.String("filename", file.Name),
			zap.Error(err))
	} else {
		if profile.Profile.DisplayName != "" {
			user = profile.Profile.DisplayName
		} else if profile.Profile.RealName != "" {
			user = profile.Profile.RealName
		} else {
			user = profile.Name
		}
	}

	Logger.Info("File upload detected", zap.String("username", user), zap.String("filename", file.Name))
	if buf, err := fileBytes(file); err != nil {
		Logger.Error(
			"error downloading file",
			zap.Error(err),
			zap.String("username", user),
			zap.String("filename", file.Name))
	} else {
		path := fmt.Sprintf("%s/%s", CdnPath, time.Now().Format("2006/01/02/15"))
		if err := os.MkdirAll(path, 0755); err != nil {
			Logger.Error(
				"error making cdn path",
				zap.String("path", path),
				zap.String("username", user),
				zap.String("filename", file.Name))
			return false
		}
		part := &url.URL{Path: file.Name}
		urlPath := fmt.Sprintf("%s/%s-%s", path, file.ID, part.String())
		path = fmt.Sprintf("%s/%s-%s", path, file.ID, file.Name)
		if fp, err := os.Create(path); err != nil {
			Logger.Error(
				"error creating cdn file",
				zap.String("path", path),
				zap.String("username", user),
				zap.String("filename", file.Name))
			return false
		} else {
			if _, err := fp.Write(buf); err != nil {
				fp.Close()
				Logger.Error(
					"error writing to cdn file",
					zap.String("path", path),
					zap.String("username", user),
					zap.String("filename", file.Name))
				return false
			}
			fp.Close()
			fileURL := CdnPrefix + urlPath[len(CdnPath):]

			//delete file and message
			if err := rtm.DeleteFile(file.ID); err != nil {
				Logger.Error(fmt.Sprintf("unable to delete file - %s", err.Error()),
					zap.String("file", file.ID))
			}
			if _, _, err := rtm.DeleteMessage(m.Channel, m.Timestamp); err != nil {
				if err.Error() != "message_not_found" { // message not found means the file deletion deleted the message
					Logger.Error(fmt.Sprintf("unable to delete message - %s", err.Error()),
						zap.String("username", user),
						zap.String("channel", m.Channel),
						zap.String("messageTime", m.Timestamp),
						zap.Error(err))
				}
			}

			if isImage.MatchString(strings.ToLower(file.Name)) {
				for i := 0; i < 5; i++ {
					_, _, err := rtm.PostMessage(
						m.Channel,
						"",
						slack.PostMessageParameters{
							AsUser:      true,
							UnfurlLinks: true,
							UnfurlMedia: true,
							IconEmoji:   ":paperclip:",
							Attachments: []slack.Attachment{
								slack.Attachment{
									AuthorName: fmt.Sprintf("%s uploaded %s", user, file.Name),
									Pretext:    m.Text,
									ImageURL:   fileURL,
								},
							},
							ThreadTimestamp: m.ThreadTimestamp,
						})
					if err != nil {
						Logger.Error(
							"Failed postting cdn link back to slack",
							zap.String("username", user),
							zap.String("filename", file.Name),
							zap.String("url", fileURL))
					} else {
						break
					}
					time.Sleep(time.Second * time.Duration(i))
				}
			} else {
				rtm.SendMessage(&slack.OutgoingMessage{
					ID:              int(time.Now().UnixNano()),
					Channel:         m.Channel,
					Text:            fmt.Sprintf("%s uploaded the file *%s*\n%s", user, file.Title, fileURL),
					Type:            "message",
					ThreadTimestamp: m.ThreadTimestamp,
				})
			}
			Logger.Info("saved CDN file", zap.String("url", fileURL), zap.Int("size", len(buf)))
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
	if buf, err := fileBytes(&m.Files[0]); err != nil {
		Logger.Info("error downloading file", zap.Error(err))
	} else {
		path := fmt.Sprintf("%s/%s", CdnPath, time.Now().Format("2006/01/02/15"))
		if err := os.MkdirAll(path, 0755); err != nil {
			Logger.Error("error making cdn path", zap.String("path", path))
			return false
		}
		part := &url.URL{Path: m.Files[0].Name}
		urlPath := fmt.Sprintf("%s/%s-%s", path, m.Files[0].ID, part.String())
		path = fmt.Sprintf("%s/%s-%s", path, m.Files[0].ID, m.Files[0].Name)
		if fp, err := os.Create(path); err != nil {
			Logger.Error("error creating cdn file", zap.String("path", path))
			return false
		} else {
			if _, err := fp.Write(buf); err != nil {
				fp.Close()
				Logger.Error("error writing to cdn file", zap.String("path", path))
				return false
			}
			fp.Close()
			fileURL := CdnPrefix + urlPath[len(CdnPath):]
			rtm.DeleteFile(m.Files[0].ID)
			rtm.SendMessage(&slack.OutgoingMessage{
				ID:      int(time.Now().UnixNano()),
				Channel: m.Channel,
				Text:    fmt.Sprintf("Thanks for sending me the file instead of uploading it to a channel or group. You can paste the following link anywhere you want to show the file to others! ```%s```", fileURL),
				Type:    "message",
			})
			Logger.Info("saved CDN file", zap.String("url", fileURL), zap.Int("size", len(buf)))
		}

	}
	return true
}
