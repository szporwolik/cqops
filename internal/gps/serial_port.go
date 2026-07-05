package gps

import (
	"fmt"

	"go.bug.st/serial"
)

// serialPortImpl implements serialPort using go.bug.st/serial.
type serialPortImpl struct {
	port serial.Port
}

func openSerial(portName string, baud int, dtr, rts bool) (serialPort, error) {
	mode := &serial.Mode{
		BaudRate: baud,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}
	p, err := serial.Open(portName, mode)
	if err != nil {
		return nil, fmt.Errorf("GPS: cannot open %s: %w", portName, err)
	}
	if dtr {
		if err := p.SetDTR(true); err != nil {
			p.Close()
			return nil, fmt.Errorf("GPS: DTR failed: %w", err)
		}
	}
	if rts {
		if err := p.SetRTS(true); err != nil {
			p.Close()
			return nil, fmt.Errorf("GPS: RTS failed: %w", err)
		}
	}
	return &serialPortImpl{port: p}, nil
}

func (s *serialPortImpl) Close() error {
	return s.port.Close()
}

func (s *serialPortImpl) Read(p []byte) (int, error) {
	return s.port.Read(p)
}
