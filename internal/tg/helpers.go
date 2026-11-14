package tg

import (
	"fmt"

	"github.com/gotd/td/tg"
)

func getTitle(source any) string {
	var name string
	switch u := source.(type) {
	case *tg.User:
		name = u.FirstName
		if u.LastName != "" {
			name += " " + u.LastName
		}

	case *tg.Chat:
		name = u.Title
	case *tg.Channel:
		name = u.Title
	}

	return name
}

func getUsername(source any) string {
	var username string
	switch u := source.(type) {
	case *tg.User:
		username = u.Username
	case *tg.Chat:
		username = fmt.Sprintf("cht[%d]", u.ID)
	case *tg.Channel:
		username = u.Username
		if username == "" {
			username = fmt.Sprintf("chn[%d:%d]", u.ID, u.AccessHash)
		}
	}

	return username
}
