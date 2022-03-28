package server

import (
	"bytes"
	"crypto/sha1"
	_ "embed"
	"encoding/base64"
	"html/template"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/clementd64/ebook-server/pkg/epub"
	"github.com/gofiber/fiber/v2"
)

//go:embed index.html
var browserTemplate string

//go:embed kindle.html
var kindleTemplate string

func New(basePath string) (*fiber.App, error) {
	ebook := []*epub.Epub{}
	ebookCategory := map[string][]*epub.Epub{}
	index := map[string]*epub.Epub{}

	if err := filepath.Walk(basePath, func(p string, f os.FileInfo, err error) error {
		if f.IsDir() || path.Ext(p) != ".epub" {
			return nil
		}

		e, err := epub.Open(p)
		if err != nil {
			return err
		}

		ebook = append(ebook, e)
		if _, ok := ebookCategory[e.Serie]; !ok {
			ebookCategory[e.Serie] = []*epub.Epub{}
		}
		ebookCategory[e.Serie] = append(ebookCategory[e.Serie], e)

		if e.Identifier == nil || len(e.Identifier) == 0 {
			hasher := sha1.New()
			hasher.Write([]byte(p))
			id := "internal:" + base64.RawURLEncoding.EncodeToString(hasher.Sum(nil))

			e.Identifier = []string{id}
			index[id] = e
		} else {
			for _, id := range e.Identifier {
				index[id] = e
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		templateData := browserTemplate
		if strings.Contains(c.Get("User-Agent"), "Kindle") {
			templateData = kindleTemplate
		}

		t := template.Must(template.New("").Parse(templateData))

		var buffer bytes.Buffer
		if err := t.Execute(&buffer, map[string]interface{}{
			"Ebook":    ebook,
			"Category": ebookCategory,
		}); err != nil {
			log.Print(err)
			return c.SendStatus(500)
		}

		c.Set("Content-Type", "text/html")
		return c.Send(buffer.Bytes())
	})

	app.Get("/json", func(c *fiber.Ctx) error {
		return c.JSON(ebook)
	})

	app.Get("/cover/:id/", func(c *fiber.Ctx) error {
		e, ok := index[c.Params("id")]
		if !ok {
			return c.SendStatus(404)
		}

		data, mime, err := e.GetCover()
		if err != nil {
			log.Print(err)
			return c.SendStatus(500)
		}

		c.Set("Content-Type", mime)
		return c.Send(data)
	})

	app.Get("/download/:id/:name.epub", func(c *fiber.Ctx) error {
		e, ok := index[c.Params("id")]
		if !ok {
			return c.SendStatus(404)
		}

		c.Set("Content-Type", "application/epub+zip")
		return c.SendFile(e.Filename)
	})

	app.Get("/download/:id/:name.azw", func(c *fiber.Ctx) error {
		e, ok := index[c.Params("id")]
		if !ok {
			return c.SendStatus(404)
		}

		tmp := "/tmp/" + strconv.FormatUint(uint64(rand.Uint32()), 10) + ".azw3"
		defer os.Remove(tmp)

		if err := exec.Command("ebook-convert", e.Filename, tmp).Run(); err != nil {
			log.Print(err)
			return c.SendStatus(500)
		}

		c.Set("Content-Type", "application/octet-stream")
		return c.SendFile(tmp)
	})

	return app, nil
}
