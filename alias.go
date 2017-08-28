// Type aliases were introduced in Go 1.9
// +build go1.9

package selenium

import "github.com/tebeka/selenium/seleniumlog"

// The following were migrated to the log package on 17/18 July 2017, and then
// to the seleniumlog package on 28 August 2017.
type LogMessage = seleniumlog.Message
type LogType = seleniumlog.Type
