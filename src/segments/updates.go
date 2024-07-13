package segments

import (
	"bufio"
	"fmt"
	"github.com/jandedobbeleer/oh-my-posh/src/properties"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"
	"os"
	"strconv"
)

type Updates struct {
	props properties.Properties
	env   runtime.Environment

	RebootIcon       string
	UpdatesIcon      string
	RebootRequired   string
	UpdatesAvailable string
}

const (
	RebootIcon properties.Property = "reboot"
	UpdateIcon properties.Property = "update"
	// RebootRequired   properties.Property = "reboot"
	// UpdatesAvailable properties.Property = "updates"

	RebootIndicator string = "/var/run/reboot-required"
	UpdateIndicator string = "/var/lib/update-notifier/updates-available"
)

func (e *Updates) Template() string {
	return "{{ .RebootRequired }}{{ .UpdatesAvailable }}"
}

func (e *Updates) Enabled() bool {
	e.RebootIcon = ""
	if _, err := os.Stat(RebootIndicator); err == nil {
		e.RebootIcon = e.props.GetString(RebootIcon, "\u27F3") // xEAD2
		e.RebootRequired = fmt.Sprintf(" %s", e.UpdatesIcon)
	}
	e.UpdatesIcon = ""
	e.UpdatesAvailable = ""
	file, err := os.Open(UpdateIndicator)
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanWords)
		count := 0
		for scanner.Scan() {
			word := scanner.Text()
			if n, err := strconv.Atoi(word); err == nil {
				count += n
				// fmt.Println(word, n, count)
			}
		}
		if count > 0 {
			e.UpdatesIcon = e.props.GetString(UpdateIcon, "\uEB42")
			e.UpdatesAvailable = fmt.Sprintf(" %s %d", e.UpdatesIcon, count)
		}
	}

	// if e.props.GetBool(properties.AlwaysEnabled, false) {
	//	return true
	//}
	return true
}

func (e *Updates) Init(props properties.Properties, env runtime.Environment) {
	e.props = props
	e.env = env
}
