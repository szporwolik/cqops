package rotor

// Status holds a rotor position snapshot.
type Status struct {
	Connected bool
	Azimuth   float64 // degrees
	Elevation float64 // degrees
}
