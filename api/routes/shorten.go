package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/DraftTin/URLShortener-go-fiber-redis/api/database"
	"github.com/DraftTin/URLShortener-go-fiber-redis/api/helpers"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

type request struct {
	URL         string        `json:"url"`
	CustomShort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	URL            string        `json:"url"`
	CustomShort    string        `json:"short"`
	Expiry         time.Duration `json:"expiry"`
	XRateRemaining int           `json:"rate_limit"`
	XRateLimitRest time.Duration `json:"rate_limit_reset"`
}

func ShortenURL(c *fiber.Ctx) error {
	body := new(request)
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	// implement rate limiting
	client := database.CreateClient(1)
	defer client.Close()
	val, err := client.Get(database.Ctx, c.IP()).Result()
	if err == redis.Nil {
		_ = client.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := client.TTL(database.Ctx, c.IP()).Result()
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":            "Rate limit exceed",
				"rate_limit_reset": limit / time.Minute,
			})
		}
	}

	// check if it's an actual url
	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error: Invalid URL"})
	}
	// check for domain error
	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Invalid URL"})
	}

	// enforce https, ssl
	body.URL = helpers.EnforceHTTP(body.URL)

	client.Decr(database.Ctx, c.IP())
}
