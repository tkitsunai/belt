package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/gofiber/fiber/v2"
	"log"
	"net/http"
)

func main() {
	db, err := bolt.Open("shorturls.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := fiber.New()

	app.Post("/shorten", func(c *fiber.Ctx) error {
		var requestData struct {
			URL string `json:"url"`
		}
		if err := c.BodyParser(&requestData); err != nil {
			return err
		}

		uniqueID := generateUniqueID(requestData.URL)

		err := db.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte("shorturls"))
			if err != nil {
				return err
			}
			return b.Put([]byte(uniqueID), []byte(requestData.URL))
		})
		if err != nil {
			return err
		}

		shortURL := fmt.Sprintf("http://localhost:3000/%s", uniqueID)

		responseData := struct {
			ShortURL    string `json:"short_url"`
			OriginalURL string `json:"original_url"`
		}{ShortURL: shortURL, OriginalURL: requestData.URL}

		return c.Status(http.StatusCreated).JSON(responseData)
	})

	app.Get("/:hash", func(c *fiber.Ctx) error {
		hash := c.Params("hash")
		var originalURL string
		err := db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("shorturls"))
			if b == nil {
				return fiber.ErrNotFound
			}
			originalURL = string(b.Get([]byte(hash)))
			if originalURL == "" {
				return fiber.ErrNotFound
			}
			return nil
		})
		if err != nil {
			return err
		}

		err = c.Redirect(originalURL, http.StatusMovedPermanently)
		if err != nil {
			return err
		}

		return nil
	})

	app.Delete("/:hash", func(c *fiber.Ctx) error {
		hash := c.Params("hash")

		err := db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("shorturls"))
			if b == nil {
				return fiber.ErrNotFound
			}
			return b.Delete([]byte(hash))
		})
		if err != nil {
			return err
		}

		return c.JSON(struct {
			Message string `json:"message"`
		}{Message: "URL deleted successfully"})
	})

	log.Fatal(app.Listen(":3000"))
}

func generateUniqueID(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])[:8]
}
