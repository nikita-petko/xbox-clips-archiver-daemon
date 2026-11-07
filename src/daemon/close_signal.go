package daemon

import "os"

// CloseSignal is the signal used to close the app
// either from an OS signal or manual signal.
var CloseSignal chan os.Signal
