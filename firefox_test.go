package selenium

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/blang/semver"
	"github.com/tebeka/selenium/firefox"
)

var (
	selenium2Path          = flag.String("selenium2_path", "vendor/selenium-server-standalone-2.53.1.jar", "The path to the Selenium 2 server JAR. If empty or the file is not present, Firefox tests on Selenium 2 will not be run.")
	firefoxBinarySelenium2 = flag.String("firefox_binary_for_selenium2", "vendor/firefox-47/firefox", "The name of the Firefox binary for Selenium 2 tests or the path to it. If the name does not contain directory separators, the PATH will be searched.")

	selenium3Path          = flag.String("selenium3_path", "vendor/selenium-server-standalone-3.4.jar", "The path to the Selenium 3 server JAR. If empty or the file is not present, Firefox tests using Selenium 3 will not be run.")
	firefoxBinarySelenium3 = flag.String("firefox_binary_for_selenium3", "vendor/firefox-nightly/firefox", "The name of the Firefox binary for Selenium 3 tests or the path to it. If the name does not contain directory separators, the PATH will be searched.")
	geckoDriverPath        = flag.String("geckodriver_path", "vendor/geckodriver-v0.16.1-linux64", "The path to the geckodriver binary. If empty of the file is not present, the Geckodriver tests will not be run.")
)

func TestFirefoxSelenium2(t *testing.T) {
	if *useDocker {
		t.Skip("Skipping tests because they will be run under a Docker container")
	}
	if _, err := os.Stat(*selenium2Path); err != nil {
		t.Skipf("Skipping Firefox tests using Selenium 2 because Selenium WebDriver JAR not found at path %q", *selenium2Path)
	}
	runFirefoxTests(t, *selenium2Path, config{
		seleniumVersion: semver.MustParse("2.0.0"),
		path:            *firefoxBinarySelenium2,
	})
}

func TestFirefoxSelenium3(t *testing.T) {
	if *useDocker {
		t.Skip("Skipping tests because they will be run under a Docker container")
	}
	if _, err := os.Stat(*selenium3Path); err != nil {
		t.Skipf("Skipping Firefox tests using Selenium 3 because Selenium WebDriver JAR not found at path %q", *selenium3Path)
	}
	if _, err := os.Stat(*geckoDriverPath); err != nil {
		t.Skipf("Skipping Firefox tests on Selenium 3 because geckodriver binary %q not found", *geckoDriverPath)
	}

	runFirefoxTests(t, *selenium3Path, config{
		seleniumVersion: semver.MustParse("3.0.0"),
		serviceOptions:  []ServiceOption{GeckoDriver(*geckoDriverPath)},
		path:            *firefoxBinarySelenium3,
	})
}

func TestFirefoxGeckoDriver(t *testing.T) {
	if *useDocker {
		t.Skip("Skipping tests because they will be run under a Docker container")
	}
	if _, err := os.Stat(*geckoDriverPath); err != nil {
		t.Skipf("Skipping Firefox tests on Selenium 3 because geckodriver binary %q not found", *geckoDriverPath)
	}

	runFirefoxTests(t, *geckoDriverPath, config{
		path: *firefoxBinarySelenium3,
	})
}

func runFirefoxTests(t *testing.T, webDriverPath string, c config) {
	c.browser = "firefox"

	if s, err := os.Stat(c.path); err != nil || !s.Mode().IsRegular() {
		if path, err := exec.LookPath(c.path); err == nil {
			c.path = path
		} else {
			t.Skipf("Skipping Firefox tests because binary %q not found", c.path)
		}
	}

	if *startFrameBuffer {
		c.serviceOptions = append(c.serviceOptions, StartFrameBuffer())
	}
	if testing.Verbose() {
		SetDebug(true)
		c.serviceOptions = append(c.serviceOptions, Output(os.Stderr))
	}

	port, err := pickUnusedPort()
	if err != nil {
		t.Fatalf("pickUnusedPort() returned error: %v", err)
	}

	var s *Service
	if c.seleniumVersion.Major == 0 {
		s, err = NewGeckoDriverService(webDriverPath, port, c.serviceOptions...)
	} else {
		s, err = NewSeleniumService(webDriverPath, port, c.serviceOptions...)
	}
	if err != nil {
		t.Fatalf("Error starting the WebDriver server with binary %q: %v", webDriverPath, err)
	}

	if c.seleniumVersion.Major == 0 {
		c.addr = fmt.Sprintf("http://127.0.0.1:%d", port)
	} else {
		c.addr = fmt.Sprintf("http://127.0.0.1:%d/wd/hub", port)
	}

	runTests(t, c)

	// Firefox-specific tests.
	t.Run("Preferences", runTest(testFirefoxPreferences, c))
	t.Run("Profile", runTest(testFirefoxProfile, c))

	if err := s.Stop(); err != nil {
		t.Fatalf("Error stopping the Selenium service: %v", err)
	}
}

func testFirefoxPreferences(t *testing.T, c config) {
	if c.seleniumVersion.Major == 2 {
		t.Skip("This test is known to fail for Selenium 2 and Firefox 47.")
	}
	caps := newTestCapabilities(t, c)
	f, ok := caps[firefox.CapabilitiesKey].(firefox.Capabilities)
	if !ok || f.Prefs == nil {
		f.Prefs = make(map[string]interface{})
	}
	f.Prefs["browser.startup.homepage"] = serverURL
	f.Prefs["browser.startup.page"] = "1"
	caps.AddFirefox(f)

	wd := &remoteWD{
		capabilities: caps,
		urlPrefix:    c.addr,
	}
	defer func() {
		if err := wd.Quit(); err != nil {
			t.Errorf("wd.Quit() returned error: %v", err)
		}
	}()

	if _, err := wd.NewSession(); err != nil {
		t.Fatalf("error in new session - %s", err)
	}

	u, err := wd.CurrentURL()
	if err != nil {
		t.Fatalf("wd.Current() returned error: %v", err)
	}
	if u != serverURL+"/" {
		t.Fatalf("wd.Current() = %q, want %q", u, serverURL+"/")
	}
}

func testFirefoxProfile(t *testing.T, c config) {
	if c.seleniumVersion.Major == 2 {
		t.Skip("This test is known to fail for Selenium 2 and Firefox 47.")
	}
	caps := newTestCapabilities(t, c)
	f := caps[firefox.CapabilitiesKey].(firefox.Capabilities)
	const path = "testing/firefox-profile"
	if err := f.SetProfile(path); err != nil {
		t.Fatalf("f.SetProfile(%q) returned error: %v", path, err)
	}
	caps.AddFirefox(f)

	wd := &remoteWD{
		capabilities: caps,
		urlPrefix:    c.addr,
	}
	if _, err := wd.NewSession(); err != nil {
		t.Fatalf("wd.NewSession() returned error: %v", err)
	}
	defer quitRemote(t, wd)

	u, err := wd.CurrentURL()
	if err != nil {
		t.Fatalf("wd.Current() returned error: %v", err)
	}
	const wantURL = "about:config"
	if u != wantURL {
		t.Fatalf("wd.Current() = %q, want %q", u, wantURL)
	}

	// Test that the old Firefox profile location gets migrated for W3C
	// compatibility.
	caps = newW3CCapabilities(map[string]interface{}{"firefox_profile": "base64-encoded Firefox profile goes here"})
	fmt.Printf("%v", caps)
	f = caps["alwaysMatch"].(Capabilities)[firefox.CapabilitiesKey].(firefox.Capabilities)
	if f.Profile == "" {
		t.Fatalf("Capability 'firefox_profile' was not migrated to 'moz:firefoxOptions.profile': %+v", caps)
	}
}
