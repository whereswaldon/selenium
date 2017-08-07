package selenium

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/tebeka/selenium/chrome"
)

var (
	chromeDriverPath = flag.String("chrome_driver_path", "vendor/chromedriver-linux64-2.30", "The path to the ChromeDriver binary. If empty of the file is not present, Chrome tests will not be run.")
	chromeBinary     = flag.String("chrome_binary", "vendor/chrome-linux/chrome", "The name of the Chrome binary or the path to it. If name is not an exact path, the PATH will be searched.")
)

func TestChrome(t *testing.T) {
	if *useDocker {
		t.Skip("Skipping Chrome tests because they will be run under a Docker container")
	}
	if _, err := os.Stat(*chromeBinary); err != nil {
		path, err := exec.LookPath(*chromeBinary)
		if err != nil {
			t.Skipf("Skipping Chrome tests because binary %q not found", *chromeBinary)
		}
		*chromeBinary = path
	}
	if _, err := os.Stat(*chromeDriverPath); err != nil {
		t.Skipf("Skipping Chrome tests because ChromeDriver not found at path %q", *chromeDriverPath)
	}

	var opts []ServiceOption
	if *startFrameBuffer {
		opts = append(opts, StartFrameBuffer())
	}
	if testing.Verbose() {
		SetDebug(true)
		opts = append(opts, Output(os.Stderr))
	}

	port, err := pickUnusedPort()
	if err != nil {
		t.Fatalf("pickUnusedPort() returned error: %v", err)
	}

	s, err := NewChromeDriverService(*chromeDriverPath, port, opts...)
	if err != nil {
		t.Fatalf("Error starting the ChromeDriver server: %v", err)
	}
	c := config{
		addr:    fmt.Sprintf("http://127.0.0.1:%d/wd/hub", port),
		browser: "chrome",
		path:    *chromeBinary,
	}

	runTests(t, c)

	// Chrome-specific tests.
	t.Run("Extension", runTest(testChromeExtension, c))

	if err := s.Stop(); err != nil {
		t.Fatalf("Error stopping the ChromeDriver service: %v", err)
	}
}

func testChromeExtension(t *testing.T, c config) {
	caps := newTestCapabilities(t, c)
	co := caps[chrome.CapabilitiesKey].(chrome.Capabilities)
	const path = "testing/chrome-extension/css_page_red"
	if err := co.AddUnpackedExtension(path); err != nil {
		t.Fatalf("co.AddExtension(%q) returned error: %v", path, err)
	}
	caps[chrome.CapabilitiesKey] = co

	wd, err := NewRemote(caps, c.addr)
	if err != nil {
		t.Fatalf("NewRemote(_, _) returned error: %v", err)
	}
	defer wd.Quit()

	if err := wd.Get(serverURL); err != nil {
		t.Fatalf("wd.Get(%q) returned error: %v", serverURL, err)
	}
	e, err := wd.FindElement(ByCSSSelector, "body")
	if err != nil {
		t.Fatalf("error finding body: %v", err)
	}

	const property = "background-color"
	color, err := e.CSSProperty(property)
	if err != nil {
		t.Fatalf(`e.CSSProperty(%q) returned error: %v`, property, err)
	}

	const wantColor = "rgba(255, 0, 0, 1)"
	if color != wantColor {
		t.Fatalf("body background has color %q, want %q", color, wantColor)
	}
}
