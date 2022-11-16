package segments

import (
	// "bufio"
	"encoding/json"
	"fmt"
	"io/ioutil" //nolint:staticcheck,nolintlint
	"oh-my-posh/platform"
	"oh-my-posh/properties"
	"path/filepath"
	"strconv"
	"strings"
)

const sysfs = "/sys/class/hwmon"
const (
	// CacheKeyResponse key used when caching the response
	CacheKeyHWMonitors string = "hwmons"
)

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
	env   platform.Environment

	FanIcon     string
	TempIcon    string
	CelsiusIcon string
	FanSpeed    string
	Temperature string
	Level       int
}

const (
	FanIcon     properties.Property = "fan_icon"
	TempIcon    properties.Property = "temp_icon"
	CelsiusIcon properties.Property = "celsius_icon"
	FanSensors  properties.Property = "fans"
	TempSensors properties.Property = "thermals"
)

var ErrNotFound = fmt.Errorf("Not found")
var NoHWMonitorsError = fmt.Errorf("Hardware Monitor not found")

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
func getMonitorName(path string) (name string, err error) {
	nameBytes, err := ioutil.ReadFile(filepath.Join(path, "name"))
	return string(nameBytes[:len(nameBytes)-1]), err
}

func getHWMonitorFiles() (map[string]string, error) {
	files, err := ioutil.ReadDir(sysfs)
	if err != nil {
		return nil, err
	}
	bNames := map[string]string{}
	for _, file := range files {
		fn := file.Name()
		path := filepath.Join(sysfs, fn)
		if name, err := getMonitorName(path); err == nil {
			bNames[name] = path
		}
	}
	if len(bNames) == 0 {
		return nil, NoHWMonitorsError
	}
	// fmt.Println(bNames)
	return bNames, nil
}

func (e *Sensors) Template() string {
	return "{{ .FanSpeed }}{{ .Temperature }}" // "°C"
}

type Response struct {
	HWMon map[string]string `json:"monitors"`
}

func (e *Sensors) getResult() (*Response, error) {
	cacheTimeout := e.props.GetInt(properties.CacheTimeout, properties.DefaultCacheTimeout)
	response := new(Response)
	if cacheTimeout > 0 {
		// check if data stored in cache
		val, found := e.env.Cache().Get(CacheKeyHWMonitors)
		// we got something from the cache
		if found {
			err := json.Unmarshal([]byte(val), response)
			if err != nil {
				return nil, err
			}
			return response, nil
		}
	}
	response.HWMon, _ = getHWMonitorFiles()
	body, _ := json.Marshal(response)
	if cacheTimeout > 0 {
		// persist new forecasts in cache
		e.env.Cache().Set(CacheKeyHWMonitors, string(body), cacheTimeout)
	}
	return response, nil
}

func (e *Sensors) Enabled() bool {
	e.FanIcon = ""
	e.TempIcon = ""
	e.CelsiusIcon = ""
	e.FanSpeed = ""
	e.Level = 0

	var monitors = map[string]string{}
	q, err := e.getResult()
	if err == nil {
		if len(q.HWMon) > 0 {
			monitors = q.HWMon
		}
	}
	if len(monitors) == 0 {
		monitors, _ = getHWMonitorFiles()
	}

	var speed int64 = 0
	for _, fan := range e.props.GetStringArray(FanSensors, []string{}) {
		fs_spec := strings.Split(fan, ":")
		if len(fs_spec) == 2 {
			if path, found := monitors[fs_spec[0]]; found {
				// fmt.Println(path)
				if s, err := readInt(path, fs_spec[1]); err == nil {
					speed = speed + s
				}
			}
		}
	}
	// fmt.Println(speed)
	temp := 0.0
	crit := 100.0
	for _, thermal := range e.props.GetStringArray(TempSensors, []string{}) {
		ts_spec := strings.Split(thermal, ":")
		if len(ts_spec) >= 2 {
			if path, found := monitors[ts_spec[0]]; found {
				if t, err := readFloat(path, ts_spec[1]); err == nil {
					if t > temp {
						temp = t
					}
					if len(ts_spec) == 3 {
						if c, err := readFloat(path, ts_spec[2]); err == nil {
							if c < crit {
								crit = c
							}
						}
					}
				}
			}
		}
	}
	e.FanIcon = e.props.GetString(FanIcon, "")
	e.TempIcon = e.props.GetString(TempIcon, "")
	e.CelsiusIcon = e.props.GetString(CelsiusIcon, "")
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
	e.Temperature = fmt.Sprintf("%c %.0f%s", icons[level], temp, e.CelsiusIcon)
	// if e.props.GetBool(properties.AlwaysEnabled, false) {
	//	return true
	//}
	return true
}

func (e *Sensors) Init(props properties.Properties, env platform.Environment) {
	e.props = props
	e.env = env
}
