package tools

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/eduardooliveira/stLib/core/models"
	"github.com/eduardooliveira/stLib/core/utils"
)

func DownloadAsset(name string, project *models.Project, client *http.Client, req *http.Request) (*models.ProjectAsset, []*models.ProjectAsset, error) {
	out, err := os.Create(utils.ToLibPath(fmt.Sprintf("%s/%s", project.FullPath(), name)))
	if err != nil {
		return nil, nil, err
	}
	defer out.Close()

	log.Println("Downloading: ", name)

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return models.NewProjectAsset(name, project, out)
}