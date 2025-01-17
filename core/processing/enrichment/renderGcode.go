package enrichment

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/eduardooliveira/stLib/core/system"
	"github.com/eduardooliveira/stLib/core/utils"
)

type gcodeRenderer struct{}

func NewGCodeRenderer() *gcodeRenderer {
	return &gcodeRenderer{}
}

type tmpImg struct {
	Height int
	Width  int
	Data   []byte
}

func (g *gcodeRenderer) Render(job Enrichable) (string, error) {
	imgName := fmt.Sprintf("%s.r.png", job.GetAsset().ID)
	imgPath := utils.ToAssetsPath(job.GetAsset().ProjectUUID, imgName)
	if _, err := os.Stat(imgPath); err == nil {
		return imgName, errors.New("already exists")
	}
	system.Publish("render", job.GetAsset().Name)

	path := utils.ToLibPath(path.Join(job.GetProject().FullPath(), job.GetAsset().Name))
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	image := &tmpImg{
		Height: 0,
		Width:  0,
	}

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		if strings.HasPrefix(strings.TrimSpace(scanner.Text()), ";") {
			line := strings.Trim(scanner.Text(), " ;")

			if strings.HasPrefix(line, "thumbnail begin") {

				header := strings.Split(line, " ")
				length, err := strconv.Atoi(header[3])
				if err != nil {
					return "", err
				}
				i, err := g.parseThumbnail(scanner, header[2], length)
				if err != nil {
					return "", err
				}
				if i.Width > image.Width || i.Height > image.Height {
					image = i
				}

			}

		}
	}

	if err := scanner.Err(); err != nil {
		return "", errors.Join(err, errors.New("error reading gcode"))
	}

	if image.Data != nil {

		utils.CreateAssetsFolder(job.GetProject().UUID)

		h := sha1.New()
		_, err = h.Write(image.Data)
		if err != nil {
			return "", err
		}

		f, err := g.storeImage(image, imgPath)
		if err != nil {
			return "", err
		}
		defer f.Close()
		return imgName, nil

	}
	return "", errors.New("no thumbnail found")
}

func (g *gcodeRenderer) parseThumbnail(scanner *bufio.Scanner, size string, length int) (*tmpImg, error) {
	sb := strings.Builder{}
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " ;")
		if strings.HasPrefix(line, "thumbnail end") {
			break
		}
		sb.WriteString(line)

	}
	if sb.Len() != length {
		return nil, errors.New("thumbnail length mismatch")
	}

	b, err := base64.StdEncoding.DecodeString(sb.String())
	if err != nil {
		return nil, err
	}

	dimensions := strings.Split(size, "x")

	img := &tmpImg{
		Data: b,
	}
	img.Height, err = strconv.Atoi(dimensions[0])
	if err != nil {
		return nil, err
	}

	img.Width, err = strconv.Atoi(dimensions[0])
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (g *gcodeRenderer) storeImage(img *tmpImg, path string) (*os.File, error) {
	i, _, err := image.Decode(bytes.NewReader(img.Data))
	if err != nil {
		return nil, err
	}
	out, _ := os.Create(path)

	err = png.Encode(out, i)

	if err != nil {
		return nil, err
	}
	return out, nil
}
