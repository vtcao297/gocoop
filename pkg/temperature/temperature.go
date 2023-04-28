//go:build linux
// +build linux

package temperature

import (
	"fmt"
	"time"
	"errors"

	"github.com/sirupsen/logrus"
	"github.com/stianeikeland/go-rpio/v4"
)

// some code taken from drr2 go-dht and tazerreloaded example for DHT22 using go-rpio.
// The reason being drr2 require compiled C code and just more complex, this rewrite
// seems to work reasonbility for my purpose

//------------------------------------------------------------------------------
// Structure
//------------------------------------------------------------------------------

// Temperature is a physical temperature(DHT11) sensor.
type SensorType int

type temperature struct {
	name           string
	sensorType	   string
	pin 		   int
	boostPerfFlag  bool
}

const (
	// DHT11 is most popular sensor
	DHT11 SensorType = iota + 1
	// DHT12 is more precise than DHT11 (has scale parts)
	DHT12
	// DHT22 is more expensive and precise than DHT11
	DHT22
	// AM2302 aka DHT22
	AM2302 = DHT22
)

// Temperature operation contract.
type Temperature interface {
	ReadTemp() (float32, float32, error)
}

//------------------------------------------------------------------------------
// Factory
//------------------------------------------------------------------------------

// NewTemperature returns a new Temperature.
func NewTemperature(name, sensorType string, pin int) Temperature {
	return &temperature{
		name:          name,
		sensorType:    sensorType,
		pin:           pin,
	}
}

// getRetryTimeout return recommended timeout necessary
// to wait before new round of data exchange.
func (v SensorType) getRetryTimeout() time.Duration {
	return 1500 * time.Millisecond
}

func readDHTXX(sensorType SensorType, pin rpio.Pin, handShakeDur time.Duration ) (temperature float32, 
	humidity float32, err error) {
	// let data line be pulled high by pull-up
	pin.Input()
	time.Sleep(1000 * time.Microsecond)
	// pull data line low for 1100 micros to signal ready to read
	pin.Output()
	pin.Low()
	time.Sleep(handShakeDur)
	// leave data line floating again, sensor writes data now
	pin.Input()
	pos := 0
	var now int64
	level := rpio.Low
	lastChange := time.Now().UnixMicro()
	syncCycles, dataCycles := make([]uint16, 50), make([]uint16, 50)

	// wait for incoming pulses from sensor until buffer is full or timeout is reached
	for {
		now = time.Now().UnixMicro()
		if pin.Read() != level {
			if level == rpio.Low {
				level = rpio.High
				// level changed to HIGH, sync cycle
				syncCycles[pos] = uint16(now - lastChange)
			} else {
				level = rpio.Low
				// level changed to LOW, data cycle
				dataCycles[pos] = uint16(now - lastChange)
				// increment position
				pos++
				if pos >= 50 {
					// buffer is full, stop reading
					break
				}
			}
			lastChange = now
		} else if now-lastChange >= 8000 {
			// pin does not change anymore, stop reading
			break
		}
	}

	// we need at least 40 pulses for a valid data packet
	if pos < 40 {
		return -1, -1, fmt.Errorf("timeout: %d packets received", pos)
	}

	// calculate average sync pulse duration
	offset := pos - 40
	var syncAverage float32 = 0
	for i := offset; i < 40; i++ {
		syncAverage += float32(syncCycles[i])
	}
	syncAverage /= 40

	// extract data bits
	data := make([]uint8, 5)
	for i := 0; i < 40; i++ {
		if dataCycles[i+offset] > uint16(syncAverage) {
			data[i/8] |= 1 << (7 - i%8)
		}
	}

	// verify checksum
	if data[4] != ((data[0] + data[1] + data[2] + data[3]) & 0xFF) {
		return -1, -1, errors.New("checksum error")
	}

	// calculate temperature and humidity
	temperature, humidity = 0.0, 0.0
	if sensorType == DHT11 {
		humidity = float32(data[0])
		temperature = float32(data[2])
	} else if sensorType == DHT12 {
		humidity = float32(data[0]) + float32(data[1])/10.0
		temperature = float32(data[2]) + float32(data[3])/10.0
		if data[3]&0x80 != 0 {
			temperature *= -1.0
		}
	} else if sensorType == DHT22 {
		humidity = (float32(data[0])*256 + float32(data[1])) / 10.0
		temperature = (float32(data[2]&0x7F)*256 + float32(data[3])) / 10.0
		if data[2]&0x80 != 0 {
			temperature *= -1.0
		}
	}

	// additional check for data correctness
	if humidity > 100.0 {
		return -1, -1, fmt.Errorf("Humidity value exceed 100%%: %v", humidity)
	} else if humidity == 0 {
		return -1, -1, fmt.Errorf("Humidity value cannot be zero")
	}

	return temperature, humidity, nil
}

// Forward turns the motor forward.
func (temp *temperature) ReadTemp() (float32, float32, error) {
	var sensorType SensorType
	var handshakeDur time.Duration
	retry := 10

	switch temp.sensorType {
	case "DHT11":
		logrus.Infof("Name = %v, Type = DHT11", temp.name)
		sensorType = DHT11
		handshakeDur = 18000 * time.Microsecond
	case "DHT12":
		logrus.Infof("Name = %v, Type = DHT12", temp.name)
		sensorType = DHT12
		handshakeDur = 200000 * time.Microsecond
	case "DHT22":
		logrus.Infof("Name = %v, Type = DHT22", temp.name)
		sensorType = DHT22
		handshakeDur = 18000 * time.Microsecond
	case "AM2302":
		logrus.Infof("Name = %v, Type = AM2302", temp.name)
		sensorType = AM2302
		handshakeDur = 18000 * time.Microsecond
	default:
		return -1, -1, fmt.Errorf("Error: not support Temp/Humidity sensor type")
	}

	// Read sensor data from specific pin, retrying 10 times in case of failure.
	err := rpio.Open()
	if err != nil {
		return -1, -1, fmt.Errorf("Error opening GPIO: %s", err)
	}
	tempPin := rpio.Pin(temp.pin)

	for {
		sensorTemp, sensorHumidity, err := readDHTXX(sensorType, tempPin, handshakeDur)
		if err != nil {
			if retry > 0 {
				logrus.Warning(err)
				retry--
				<-time.After(sensorType.getRetryTimeout())
				continue
			}
			return -1, -1, fmt.Errorf("Error accessing the Temp sensor: %s", err)
		}

		// print temperature and humidity
		logrus.Infof("Name = %v -- SensorType = %v -- Temperature = %v*F -- Humidity = %v%%",
			temp.name, sensorType, (sensorTemp * 9 / 5) + 32, sensorHumidity)

		return (sensorTemp * 9 / 5) + 32, sensorHumidity, nil
	}
}