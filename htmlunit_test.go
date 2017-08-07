package selenium

import (
	"fmt"
	"os"
	"testing"
)

func TestHTMLUnit(t *testing.T) {
	if *useDocker {
		t.Skip("Skipping tests because they will be run under a Docker container")
	}
	if _, err := os.Stat(*selenium3Path); err != nil {
		t.Skipf("Skipping Firefox tests using Selenium 3 because Selenium WebDriver JAR not found at path %q", *selenium3Path)
	}

	if testing.Verbose() {
		SetDebug(true)
	}

	c := config{
		browser: "htmlunit",
	}
	if *startFrameBuffer {
		c.serviceOptions = append(c.serviceOptions, StartFrameBuffer())
	}

	port, err := pickUnusedPort()
	if err != nil {
		t.Fatalf("pickUnusedPort() returned error: %v", err)
	}
	s, err := NewSeleniumService(*selenium3Path, port, c.serviceOptions...)
	if err != nil {
		t.Fatalf("Error starting the WebDriver server with binary %q: %v", *selenium3Path, err)
	}
	c.addr = fmt.Sprintf("http://127.0.0.1:%d/wd/hub", port)

	runTests(t, c)

	if err := s.Stop(); err != nil {
		t.Fatalf("Error stopping the Selenium service: %v", err)
	}
}
