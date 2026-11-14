package tg

import "github.com/gotd/td/telegram"

type Client struct {
	appID       int
	appHash     string
	storage     telegram.SessionStorage
	downloadDir string
}

func New(appID int, appHash string, storage telegram.SessionStorage, downloadDir string) *Client {
	return &Client{
		appID:       appID,
		appHash:     appHash,
		storage:     storage,
		downloadDir: downloadDir,
	}
}

func (c *Client) T() *telegram.Client {
	opts := telegram.Options{
		SessionStorage: c.storage,
		NoUpdates:      true,
	}
	opts, _ = telegram.OptionsFromEnvironment(opts)

	return telegram.NewClient(c.appID, c.appHash, opts)
}
