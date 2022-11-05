package segments

import (
	"bufio"
	"fmt"
	"oh-my-posh/environment"
	"oh-my-posh/properties"
	"os"
	"strconv"
)

type Updates struct {
	props properties.Properties
	env   environment.Environment

	RebootIcon       string
	UpdatesIcon      string
	UpdatesAvailable string
}

const (
	RebootRequired   properties.Property = "reboot"
	UpdatesAvailable properties.Property = "updates"

	RebootIndicator string = "/var/run/reboot-required"
	UpdateIndicator string = "/var/lib/update-notifier/updates-available"
)

func (e *Updates) Template() string {
	return " {{ .RebootIcon }} {{ .UpdatesAvailable }} "
}

func (e *Updates) Enabled() bool {
	e.RebootIcon = ""
	if _, err := os.Stat(RebootIndicator); err == nil {
		e.RebootIcon = e.props.GetString(RebootRequired, "\u27F3") // xEAD2
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
			e.UpdatesIcon = e.props.GetString(UpdatesAvailable, "\uEB42")
			e.UpdatesAvailable = fmt.Sprintf("%s%d", e.UpdatesIcon, count)
		}
	}

	// if e.props.GetBool(properties.AlwaysEnabled, false) {
	//	return true
	//}
	return true
}

func (e *Updates) Init(props properties.Properties, env environment.Environment) {
	e.props = props
	e.env = env
}
