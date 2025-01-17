package thingiverse

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/eduardooliveira/stLib/core/data/database"
	"github.com/eduardooliveira/stLib/core/downloader/tools"
	"github.com/eduardooliveira/stLib/core/entities"
	"github.com/eduardooliveira/stLib/core/runtime"
	"github.com/eduardooliveira/stLib/core/utils"
)

func Fetch(url string) error {

	if runtime.Cfg.Integrations.Thingiverse.Token == "" {
		return errors.New("missing Thingiverse api token")
	}

	r := regexp.MustCompile(`thing:(\d+)`)
	matches := r.FindStringSubmatch(url)

	if len(matches) == 0 {
		return errors.New("url doesn't match thingiverse schema")
	}

	id := matches[1]
	log.Println("Processing thing: ", id)

	httpClient := &http.Client{}

	project := entities.NewProject("CHANGE ME")

	err := fetchDetails(id, project, httpClient)
	if err != nil {
		return err
	}

	if p, err := database.GetProjectByPathAndName(project.Path, project.Name); err == nil && p.UUID != "" {
		project.UUID = p.UUID
	}

	if err = utils.CreateFolder(utils.ToLibPath(project.FullPath())); err != nil {
		log.Println("error creating project folder")
		return err
	}

	if err = utils.CreateAssetsFolder(project.UUID); err != nil {
		log.Println("error creating assets folder")
		return err
	}

	err = fetchFiles(id, project, httpClient)
	if err != nil {
		log.Println("error fetching files")
		return err
	}
	err = fetchImages(id, project, httpClient)
	if err != nil {
		log.Println("error fetching images")
		return err
	}

	project.Initialized = true

	return database.UpdateProject(project)
}

func fetchDetails(id string, project *entities.Project, httpClient *http.Client) error {
	u := &url.URL{Scheme: "https", Host: "api.thingiverse.com", Path: "/things/" + id}

	req := &http.Request{
		Method: "GET",
		URL:    u,
		Header: http.Header{
			"Authorization": []string{"Bearer " + runtime.Cfg.Integrations.Thingiverse.Token},
		},
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	thing := &Thing{}
	if err := json.NewDecoder(res.Body).Decode(thing); err != nil {
		return err
	}

	project.Name = strings.ReplaceAll(fmt.Sprintf("%d - %s", thing.ID, thing.Name), "/", "-")
	project.Description = thing.Description
	project.ExternalLink = thing.PublicURL

	for _, tag := range thing.Tags {
		project.Tags = append(project.Tags, entities.StringToTag(tag.Name))
	}

	log.Println("Downloading details for thing: ", thing.Name)

	return nil
}

func fetchFiles(id string, project *entities.Project, httpClient *http.Client) error {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "https", Host: "api.thingiverse.com", Path: "/things/" + id + "/files"},
		Header: http.Header{
			"Authorization": []string{"Bearer " + runtime.Cfg.Integrations.Thingiverse.Token},
		},
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var files []*ThingFile
	if err := json.NewDecoder(res.Body).Decode(&files); err != nil {
		return err
	}

	req.Method = "GET"

	for _, file := range files {

		req.URL, _ = url.Parse(file.DownloadURL)

		err := tools.DownloadAsset(file.Name, project, httpClient, req)
		if err != nil {
			return err
		}

	}

	log.Printf("Downloaded %d files\n", len(files))

	return nil
}

func fetchImages(id string, project *entities.Project, httpClient *http.Client) error {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "https", Host: "api.thingiverse.com", Path: "/things/" + id + "/images"},
		Header: http.Header{
			"Authorization": []string{"Bearer " + runtime.Cfg.Integrations.Thingiverse.Token},
		},
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var tImages []*ThingImage
	if err := json.NewDecoder(res.Body).Decode(&tImages); err != nil {
		return err
	}

	req.Method = "GET"

	for _, image := range tImages {

		for _, size := range image.Sizes {
			if size.Size == "large" && size.Type == "display" {

				req.URL, _ = url.Parse(size.URL)

				err := tools.DownloadAsset(image.Name, project, httpClient, req)
				if err != nil {
					return err
				}

			}
		}

	}

	log.Printf("Downloaded %d images\n", len(tImages))

	return nil
}

type ThingImage struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	URL   string `json:"url"`
	Sizes []struct {
		Type string `json:"type"`
		Size string `json:"size"`
		URL  string `json:"url"`
	} `json:"sizes"`
}

type ThingFile struct {
	ID            int           `json:"id"`
	Name          string        `json:"name"`
	Size          int           `json:"size"`
	URL           string        `json:"url"`
	PublicURL     string        `json:"public_url"`
	DownloadURL   string        `json:"download_url"`
	ThreejsURL    string        `json:"threejs_url"`
	Thumbnail     string        `json:"thumbnail"`
	DefaultImage  interface{}   `json:"default_image"`
	Date          string        `json:"date"`
	FormattedSize string        `json:"formatted_size"`
	MetaData      []interface{} `json:"meta_data"`
	DownloadCount int           `json:"download_count"`
	DirectURL     string        `json:"direct_url"`
}

type Thing struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Thumbnail string `json:"thumbnail"`
	URL       string `json:"url"`
	PublicURL string `json:"public_url"`
	Creator   struct {
		ID               int    `json:"id"`
		Name             string `json:"name"`
		FirstName        string `json:"first_name"`
		LastName         string `json:"last_name"`
		URL              string `json:"url"`
		PublicURL        string `json:"public_url"`
		Thumbnail        string `json:"thumbnail"`
		CountOfFollowers int    `json:"count_of_followers"`
		CountOfFollowing int    `json:"count_of_following"`
		CountOfDesigns   int    `json:"count_of_designs"`
		AcceptsTips      bool   `json:"accepts_tips"`
		IsFollowing      bool   `json:"is_following"`
		Location         string `json:"location"`
		Cover            string `json:"cover"`
	} `json:"creator"`
	Added        time.Time   `json:"added"`
	Modified     time.Time   `json:"modified"`
	IsPublished  int         `json:"is_published"`
	IsWip        int         `json:"is_wip"`
	IsFeatured   interface{} `json:"is_featured"`
	IsNsfw       bool        `json:"is_nsfw"`
	LikeCount    int         `json:"like_count"`
	IsLiked      bool        `json:"is_liked"`
	CollectCount int         `json:"collect_count"`
	IsCollected  bool        `json:"is_collected"`
	CommentCount int         `json:"comment_count"`
	IsWatched    bool        `json:"is_watched"`
	DefaultImage struct {
		ID    int    `json:"id"`
		URL   string `json:"url"`
		Name  string `json:"name"`
		Sizes []struct {
			Type string `json:"type"`
			Size string `json:"size"`
			URL  string `json:"url"`
		} `json:"sizes"`
		Added time.Time `json:"added"`
	} `json:"default_image"`
	Description      string `json:"description"`
	Instructions     string `json:"instructions"`
	DescriptionHTML  string `json:"description_html"`
	InstructionsHTML string `json:"instructions_html"`
	Details          string `json:"details"`
	DetailsParts     []struct {
		Type     string `json:"type"`
		Name     string `json:"name"`
		Required string `json:"required,omitempty"`
		Data     []struct {
			Content string `json:"content"`
		} `json:"data,omitempty"`
	} `json:"details_parts"`
	EduDetails        string      `json:"edu_details"`
	EduDetailsParts   interface{} `json:"edu_details_parts"`
	License           string      `json:"license"`
	AllowsDerivatives bool        `json:"allows_derivatives"`
	FilesURL          string      `json:"files_url"`
	ImagesURL         string      `json:"images_url"`
	LikesURL          string      `json:"likes_url"`
	AncestorsURL      string      `json:"ancestors_url"`
	DerivativesURL    string      `json:"derivatives_url"`
	TagsURL           string      `json:"tags_url"`
	Tags              []struct {
		Name        string `json:"name"`
		Tag         string `json:"tag"`
		URL         string `json:"url"`
		Count       int    `json:"count"`
		ThingsURL   string `json:"things_url"`
		AbsoluteURL string `json:"absolute_url"`
	} `json:"tags"`
	CategoriesURL     string      `json:"categories_url"`
	FileCount         int         `json:"file_count"`
	LayoutCount       int         `json:"layout_count"`
	LayoutsURL        string      `json:"layouts_url"`
	IsPrivate         int         `json:"is_private"`
	IsPurchased       int         `json:"is_purchased"`
	InLibrary         bool        `json:"in_library"`
	PrintHistoryCount int         `json:"print_history_count"`
	AppID             interface{} `json:"app_id"`
	DownloadCount     int         `json:"download_count"`
	ViewCount         int         `json:"view_count"`
	Education         struct {
		Grades   []interface{} `json:"grades"`
		Subjects []interface{} `json:"subjects"`
	} `json:"education"`
	RemixCount       int           `json:"remix_count"`
	MakeCount        int           `json:"make_count"`
	AppCount         int           `json:"app_count"`
	RootCommentCount int           `json:"root_comment_count"`
	Moderation       string        `json:"moderation"`
	IsDerivative     bool          `json:"is_derivative"`
	Ancestors        []interface{} `json:"ancestors"`
	CanComment       bool          `json:"can_comment"`
	TypeName         string        `json:"type_name"`
}
