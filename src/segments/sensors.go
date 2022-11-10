package segments

import (
	// "bufio"
	"fmt"
	"io/ioutil" //nolint:staticcheck,nolintlint
	"oh-my-posh/environment"
	"oh-my-posh/properties"
	"strings"

	// "os"
	"path/filepath"
	"strconv"
)

const sysfs = "/sys/class/hwmon"

/*
	_prompt_sensors() {
	    local hwmon="" mon name rest
	    [ -d /sys/class/hwmon ] || return
	    local temp= fan=0
	    for mon in /sys/class/hwmon/hwmon?; do
	        read name rest <"$mon/name"
	        case $name in
	            coretemp | cpu_thermal )
	                if [ -f "$mon/temp1_input" ]; then
	                    local temp_input="$mon/temp1_input"
	                    read temp <$temp_input
	                    temp=$(($temp / 1000))
	                fi
	                ;;
	            thinkpad)
	                if [ -f "$mon/fan1_input" ]; then
	                    local fan_input="$mon/fan1_input"
	                    read fan <$fan_input
	                fi
	                if [ -f "$mon/fan2_input" ]; then
	                    local fan_input="$mon/fan2_input" fan2=0
	                    read fan2 <$fan_input
	                    [ $fan2 -gt $fan ] && fan=$fan2
	                    unset fan2
	                fi
	                ;;
	        esac
	    done
	    local level=1
	    [ $temp -gt 30 ] && level=2
	    [ $temp -gt 45 ] && level=3
	    [ $temp -gt 60 ] && level=4
	    [ $temp -gt 75 ] && level=5

	    local msg=""
	    local fan_icon=""
	    if [ $fan -gt 0 ]; then
	        msg="$(printf '%s %d' $fan_icon $fan)"
	    fi
	    if [ $fan -gt 0 ] && [ $temp -gt 0 ]; then
	        msg="$msg "
	    fi
	    local icon=""
	    local temp_icons
	    temp_icons=(''  ''  ''  ''  '')
	    if [ $temp -gt 0 ]; then
	        msg="$msg$(printf '%s %d°C' ${temp_icons[$level]} $temp)"
	    fi
	    if [ ${BAT_POWER_NOW:=0} -gt 0 ]; then
	        local power=$((BAT_POWER_NOW / 1000000))
	        msg="$msg$(printf ' %dW' $power)"
	    fi
	    local icon_color
	    icon_color=('g' 'g' 'g' 'y' 'r')
	    local c=${icon_color[$level]}
	    _prompt_segment="\fD.\f$c.${msg}"
	}
*/

type sensor struct {
	// Current fan speed.
	FanSpeed float64
	// CPU temperate in °C.
	Temperature float64
}

type Sensors struct {
	props properties.Properties
	env   environment.Environment

	FanIcon     string
	TempIcon    string
	FanSpeed    string
	Temperature string
	Level       int
}

const (
	FanIcon    properties.Property = "fan"
	TempIcon   properties.Property = "temp"
	FanSensor  properties.Property = "fanpath"
	TempSensor properties.Property = "temppath"
)

var ErrNotFound = fmt.Errorf("Not found")

func readFloat(path, filename string) (float64, error) {
	str, err := ioutil.ReadFile(filepath.Join(path, filename))
	if err != nil {
		return 0, err
	}
	if len(str) == 0 {
		return 0, ErrNotFound
	}
	num, err := strconv.ParseFloat(string(str[:len(str)-1]), 64)
	if err != nil {
		return 0, err
	}
	return num / 1000, nil // Convert micro->milli
}

func readInt(path, filename string) (int64, error) {
	str, err := ioutil.ReadFile(filepath.Join(path, filename))
	if err != nil {
		return 0, err
	}
	if len(str) == 0 {
		return 0, ErrNotFound
	}
	num, err := strconv.ParseInt(string(str[:len(str)-1]), 10, 64)
	if err != nil {
		return 0, err
	}
	return num, err
}

func (e *Sensors) Template() string {
	return "{{ .FanSpeed }}{{ .Temperature }}°C"
}

func (e *Sensors) Enabled() bool {
	e.FanIcon = ""
	e.TempIcon = ""
	e.FanSpeed = ""
	e.Level = 0

	var speed int64 = 0
	fs := e.props.GetString(FanSensor, "")
	fs_spec := strings.Split(fs, ":")
	if len(fs_spec) == 2 {
		speed, _ = readInt("/sys/class/hwmon/hwmon"+fs_spec[0], fs_spec[1])
	}
	ts := e.props.GetString(TempSensor, "")
	ts_spec := strings.Split(ts, ":")
	temp := 0.0
	crit := 0.0
	if len(ts_spec) >= 2 {
		temp, _ = readFloat("/sys/class/hwmon/hwmon"+ts_spec[0], ts_spec[1])
		crit = 100
		if len(ts_spec) == 3 {
			crit, _ = readFloat("/sys/class/hwmon/hwmon"+ts_spec[0], ts_spec[2])
		}
	}
	e.FanIcon = e.props.GetString(FanIcon, "")
	e.TempIcon = e.props.GetString(TempIcon, "")
	if speed > 0 {
		e.FanSpeed = fmt.Sprintf("%s %d ", e.FanIcon, speed)
	}
	icons := []rune(e.TempIcon)
	level := int(float64(len(icons)) * temp / crit)
	if level < 0 {
		level = 0
	}
	if level >= len(icons) {
		level = len(icons) - 1
	}
	e.Level = level
	e.Temperature = fmt.Sprintf("%c %.0f", icons[level], temp)
	// e.SensorsAvailable = fmt.Sprintf(" %s %d", e.SensorsIcon, count)

	// if e.props.GetBool(properties.AlwaysEnabled, false) {
	//	return true
	//}
	return true
}

func (e *Sensors) Init(props properties.Properties, env environment.Environment) {
	e.props = props
	e.env = env
}
