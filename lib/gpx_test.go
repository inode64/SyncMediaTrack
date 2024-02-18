package syncmediatrack

import (
	"testing"
)

func TestReadGPX(t *testing.T) {
	filename := "../testdata/tracks/2024_01_28_08_46_Sun.gpx"
	valid := true

	err := ReadGPX(filename, valid)
	if err != nil {
		t.Errorf("Error al leer el archivo GPX válido: %v", err)
	}

	filename = "gpx_test.go"
	valid = false

	err = ReadGPX(filename, valid)
	if err == nil {
		t.Errorf("Se esperaba un error al leer el archivo GPX inválido")
	}
}
