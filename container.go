package playwrightcigo

import (
	"fmt"

	"github.com/playwright-community/playwright-go"
)

type container struct {
	terminate func()

	host  string
	ports struct {
		chromium int
		firefox  int
		webkit   int
	}
}

func (c *container) Close() {
	c.terminate()
}

func (c *container) Host() string {
	return c.host
}

func (c *container) Chromium(pw *playwright.Playwright) (playwright.Browser, error) {
	return pw.Chromium.Connect(fmt.Sprintf("ws://%s:%d/chromium", c.host, c.ports.chromium))
}

func (c *container) Firefox(pw *playwright.Playwright) (playwright.Browser, error) {
	return pw.Firefox.Connect(fmt.Sprintf("ws://%s:%d/firefox", c.host, c.ports.firefox))
}

func (c *container) WebKit(pw *playwright.Playwright) (playwright.Browser, error) {
	return pw.WebKit.Connect(fmt.Sprintf("ws://%s:%d/webkit", c.host, c.ports.webkit))
}
