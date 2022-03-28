package epub

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"path"
	"regexp"
	"strings"
)

type Package struct {
	XMLName  xml.Name `xml:"package"`
	Metadata struct {
		XMLName    xml.Name `xml:"metadata"`
		Title      string   `xml:"title"`
		Creator    []string `xml:"creator"`
		Language   string   `xml:"language"`
		Date       string   `xml:"date"`
		Publisher  string   `xml:"publisher"`
		Identifier []string `xml:"identifier"`
		Meta       []struct {
			Property string `xml:"property,attr"`
			Content  string `xml:",chardata"`
		} `xml:"meta"`
	}
	Manifest struct {
		XMLName xml.Name `xml:"manifest"`
		Item    []struct {
			ID    string `xml:"id,attr"`
			Href  string `xml:"href,attr"`
			Media string `xml:"media-type,attr"`
		} `xml:"item"`
	}
}

type Epub struct {
	Filename string
	prefix   string
	meta     Package
	Slug     string

	Title      string
	Creator    []string
	Language   string
	Date       string
	Publisher  string
	Serie      string
	Identifier []string
}

func Open(filename string) (*Epub, error) {
	e := &Epub{Filename: filename}

	if err := e.openMeta(); err != nil {
		return nil, err
	}

	e.Title = e.meta.Metadata.Title
	e.Creator = e.meta.Metadata.Creator
	e.Language = e.meta.Metadata.Language
	e.Date = e.meta.Metadata.Date
	e.Publisher = e.meta.Metadata.Publisher
	e.Identifier = e.meta.Metadata.Identifier

	baseName := strings.TrimSuffix(path.Base(filename), path.Ext(filename))

	var re = regexp.MustCompile("[^a-z0-9]+")
	e.Slug = strings.Trim(re.ReplaceAllString(strings.ToLower(baseName), "-"), "-")

	if e.Title == "" {
		e.Title = baseName
	}

	for _, meta := range e.meta.Metadata.Meta {
		if meta.Property == "belongs-to-collection" {
			e.Serie = meta.Content
		}
	}

	return e, nil
}

func (e *Epub) openMeta() error {
	zf, err := zip.OpenReader(e.Filename)
	if err != nil {
		return err
	}
	defer zf.Close()

	for _, file := range zf.File {
		if file.Name == "OEBPS/content.opf" || file.Name == "content.opf" {
			if file.Name == "OEBPS/content.opf" {
				e.prefix = "OEBPS/"
			}

			reader, err := file.Open()
			if err != nil {
				return err
			}
			defer reader.Close()

			content, err := ioutil.ReadAll(reader)
			if err != nil {
				return err
			}

			if err := xml.Unmarshal(content, &e.meta); err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *Epub) GetCover() ([]byte, string, error) {
	path := ""
	mime := ""

	for _, item := range e.meta.Manifest.Item {
		if strings.Contains(item.ID, "cover") && strings.HasPrefix(item.Media, "image") {
			path = e.prefix + item.Href
			mime = item.Media
			break
		}
	}

	if path == "" {
		return nil, "", errors.New("No cover found")
	}

	zf, err := zip.OpenReader(e.Filename)
	if err != nil {
		return nil, "", err
	}

	for _, file := range zf.File {
		if file.Name == path {
			reader, err := file.Open()
			if err != nil {
				return nil, "", err
			}
			defer reader.Close()

			content, err := ioutil.ReadAll(reader)
			if err != nil {
				return nil, "", err
			}

			return content, mime, nil
		}
	}

	return nil, "", errors.New("No cover found")
}
