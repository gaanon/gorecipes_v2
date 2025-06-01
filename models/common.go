package models

import (
	"database/sql/driver"
	"fmt"
)

// MeasurementSystem is an ENUM type for 'metric' or 'imperial' systems.
// It matches the PostgreSQL ENUM type 'measurement_system'.
type MeasurementSystem string

const (
	Metric   MeasurementSystem = "metric"
	Imperial MeasurementSystem = "imperial"
)

// String returns the string representation of MeasurementSystem.
func (ms MeasurementSystem) String() string {
	return string(ms)
}

// Scan implements the sql.Scanner interface for MeasurementSystem.
func (ms *MeasurementSystem) Scan(value interface{}) error {
	s, ok := value.([]byte) // In pgx, ENUMs often come as []byte
	if !ok {
		strVal, okStr := value.(string)
		if !okStr {
			return fmt.Errorf("failed to scan MeasurementSystem: expected string or []byte, got %T", value)
		}
		s = []byte(strVal)
	}
	*ms = MeasurementSystem(s)
	switch *ms {
	case Metric, Imperial:
		return nil
	default:
		return fmt.Errorf("invalid MeasurementSystem value: %s", s)
	}
}

// Value implements the driver.Valuer interface for MeasurementSystem.
func (ms MeasurementSystem) Value() (driver.Value, error) {
	return string(ms), nil
}
