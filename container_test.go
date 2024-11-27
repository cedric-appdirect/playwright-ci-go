package playwrightcigo

import (
	"fmt"
	"image"
	_ "image/png"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var browsers = []playwright.Browser{}

func Test_HelloWorld(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen: %v", err)
	}

	base := "http://" + l.Addr().String()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	go func() {
		err := http.Serve(l, nil)
		if err != nil {
			log.Fatalf("could not serve: %v", err)
		}
	}()

	err = wait4port(base)
	require.NoError(t, err)

	var wg sync.WaitGroup

	for idx, b := range browsers {
		wg.Add(1)
		go func(idx int, b playwright.Browser) {
			defer wg.Done()
			page, err := b.NewPage()
			defer page.Close()
			require.NoError(t, err)

			_, err = page.Goto(base)
			assert.NoError(t, err)

			testresult := filepath.Join("testdata", "failed", fmt.Sprintf("screenshot-%d.png", idx))

			page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateLoad})
			_, err = page.Screenshot(playwright.PageScreenshotOptions{
				Path: playwright.String(testresult),
			})
			assert.NoError(t, err)

			expectedSize, expectedPixels := pixels(t, filepath.Join("testdata", fmt.Sprintf("screenshot-%d.png", idx)))
			actualSize, actualPixels := pixels(t, testresult)

			assert.Equal(t, expectedPixels, actualPixels)
			assert.Equal(t, expectedSize, actualSize)
			if !t.Failed() {
				os.Remove(testresult)
			}
		}(idx, b)
	}

	wg.Wait()
}

func TestMain(m *testing.M) {
	if err := os.MkdirAll("testdata/failed", 0755); err != nil {
		log.Fatalf("could not create directory: %v", err)
	}

	if err := playwright.Install(); err != nil {
		log.Fatalf("could not install Playwright: %v", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not run Playwright: %v", err)
	}

	container, err := New(WithTimeout(5 * time.Minute))
	if err != nil {
		log.Fatalf("could not create container: %v", err)
	}

	b, err := container.Chromium(pw)
	if err != nil {
		log.Fatalf("could not connect to Chromium: %v", err)
	}
	browsers = append(browsers, b)

	b, err = container.Firefox(pw)
	if err != nil {
		log.Fatalf("could not connect to Firefox: %v", err)
	}
	browsers = append(browsers, b)

	b, err = container.WebKit(pw)
	if err != nil {
		log.Fatalf("could not connect to WebKit: %v", err)
	}
	browsers = append(browsers, b)

	code := m.Run()

	container.Close()
	_ = pw.Stop()

	os.Exit(code)
}

func pixels(t *testing.T, path string) (image.Rectangle, []uint8) {
	f, err := os.Open(path)
	assert.NoError(t, err)
	defer f.Close()

	raw, _, err := image.Decode(f)
	assert.NoError(t, err)

	var pixels []uint8
	switch raw.(type) {
	case *image.RGBA:
		pixels = raw.(*image.RGBA).Pix
	case *image.NRGBA:
		pixels = raw.(*image.NRGBA).Pix
	default:
		t.Fatalf("unsupported image type: %T", raw)
	}

	return raw.Bounds(), pixels
}
