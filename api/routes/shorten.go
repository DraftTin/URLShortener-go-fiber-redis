package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/DraftTin/URLShortener-go-fiber-redis/database"
	"github.com/DraftTin/URLShortener-go-fiber-redis/helpers"
	"github.com/asaskevich/govalidator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

	client1 := database.CreateClient(1)
	defer client1.Close()
	val, err := client1.Get(database.Ctx, c.IP()).Result()
	if err == redis.Nil {
		_ = client1.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := client1.TTL(database.Ctx, c.IP()).Result()
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":            "Rate limit exceed",
				"rate_limit_reset": limit / time.Minute,
			})
		}
	}

	// check if it's an actual url

	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid URL"})
	}
	// check for domain error

	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Invalid URL"})
	}

	// enforce https, ssl

	body.URL = helpers.EnforceHTTP(body.URL)

	// shorten the url

	id := body.CustomShort
	if body.CustomShort == "" {
		id = uuid.New().String()[:6]
	}
	client0 := database.CreateClient(0)
	defer client0.Close()

	val, _ = client0.Get(database.Ctx, id).Result()
	if val != "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "URL custom short is already in use",
		})
	}

	if body.Expiry == 0 {
		body.Expiry = 24
	}

	err = client0.Set(database.Ctx, id, body.URL, body.Expiry*3600*time.Second).Err()

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to connect the server",
		})
	}
	client1.Decr(database.Ctx, c.IP())

	val, _ = client1.Get(database.Ctx, c.IP()).Result()
	rateRemaining, _ := strconv.Atoi(val)
	ttl, _ := client1.TTL(database.Ctx, c.IP()).Result()
	ttl = ttl / time.Minute
	resp := response{
		URL:            body.URL,
		CustomShort:    os.Getenv("DOMAIN") + "/" + id,
		Expiry:         body.Expiry,
		XRateRemaining: rateRemaining,
		XRateLimitRest: ttl,
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}
