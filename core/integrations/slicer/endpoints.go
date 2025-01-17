package slicer

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/eduardooliveira/stLib/core/processing"
	"github.com/eduardooliveira/stLib/core/runtime"
	"github.com/eduardooliveira/stLib/core/state"
	"github.com/eduardooliveira/stLib/core/system"
	"github.com/labstack/echo/v4"
)

func version(c echo.Context) error {
	return c.JSON(http.StatusOK, struct {
		API    string `json:"api"`
		Server string `json:"server"`
		Text   string `json:"text"`
	}{
		API:    "xxx",
		Server: "xxx",
		Text:   "OctoPrint xx",
	})
}
func info(c echo.Context) error {
	return c.JSON(http.StatusOK, struct {
		State           string `json:"state"`
		StateMessage    string `json:"state_message"`
		Hostname        string `json:"hostname"`
		SoftwareVersion string `json:"software_version"`
		CPUInfo         string `json:"cpu_info"`
		KlipperPath     string `json:"klipper_path"`
		PythonPath      string `json:"python_path"`
		LogFile         string `json:"log_file"`
		ConfigFile      string `json:"config_file"`
	}{
		State:           "ready",
		StateMessage:    "Printer is ready",
		Hostname:        "mmp",
		SoftwareVersion: "v0.9.xxx",
		CPUInfo:         "xxx",
		KlipperPath:     "/root/klipper", // This are mock values, do not run stuff as root....
		PythonPath:      "/root/klippy-env/bin/python",
		LogFile:         "/tmp/klippy.log",
		ConfigFile:      "/root/printer.cfg",
	})
}

func upload(c echo.Context) error {

	form, err := c.MultipartForm()
	if err != nil {
		log.Println(err)
		return c.NoContent(http.StatusBadRequest)
	}

	files := form.File["file"]

	if len(files) == 0 {
		log.Println("No files")
		return c.NoContent(http.StatusBadRequest)
	}
	name := files[0].Filename

	var isNewFile = false
	if _, err := os.Stat(path.Join(runtime.GetDataPath(), "temp", name)); errors.Is(err, os.ErrNotExist) {
		isNewFile = true
	}

	// Source
	src, err := files[0].Open()
	if err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	defer src.Close()

	// Destination
	dst, err := os.Create(path.Join(runtime.GetDataPath(), "temp", name))
	if err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if isNewFile {
		tempFile, _ := processing.DiscoverTempFile(name)
		state.TempFiles[tempFile.UUID] = tempFile
	}

	system.Publish("tempfile.new", map[string]any{"name": name})
	return c.NoContent(http.StatusOK)
}
