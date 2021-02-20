package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/lithammer/shortuuid"
	"github.com/mritd/logger"
	"go.etcd.io/bbolt"
)

type DeviceInfo struct {
	DeviceKey   string `form:"device_key,omitempty" json:"device_key,omitempty" xml:"device_key,omitempty"`
	DeviceToken string `form:"device_token,omitempty" json:"device_token,omitempty" xml:"device_token,omitempty"`

	// compatible with old req
	OldDeviceKey   string `form:"key,omitempty" json:"key,omitempty" xml:"key,omitempty"`
	OldDeviceToken string `form:"devicetoken,omitempty" json:"devicetoken,omitempty" xml:"devicetoken,omitempty"`
}

const (
	bucketName = "device"
)

func init() {
	registerRoute("register", func(router *fiber.App) {
		router.Post("/register", func(c *fiber.Ctx) error { return doRegister(c, false) })

		// compatible with old requests
		router.Get("/register", func(c *fiber.Ctx) error { return doRegister(c, true) })
	})
}

func doRegister(c *fiber.Ctx, compat bool) error {
	var deviceInfo DeviceInfo
	if compat {
		if err := c.QueryParser(&deviceInfo); err != nil {
			return c.Status(400).JSON(failed(400, "request bind failed: %v", err))
		}
	} else {
		if err := c.BodyParser(&deviceInfo); err != nil {
			return c.Status(400).JSON(failed(400, "request bind failed: %v", err))
		}
	}

	if deviceInfo.DeviceKey == "" && deviceInfo.OldDeviceKey != "" {
		deviceInfo.DeviceKey = deviceInfo.OldDeviceKey
	}

	if deviceInfo.DeviceToken == "" {
		if deviceInfo.OldDeviceToken != "" {
			deviceInfo.DeviceToken = deviceInfo.OldDeviceToken
		} else {
			return c.Status(400).JSON(failed(400, "device token is empty"))
		}
	}

	err := db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))

		// If the deviceKey is empty or the corresponding deviceToken cannot be obtained from the database,
		// it is considered as a new device registration
		if deviceInfo.DeviceKey == "" || bucket.Get([]byte(deviceInfo.DeviceKey)) == nil {
			// Generate a new UUID as the deviceKey when a new device register
			deviceInfo.DeviceKey = shortuuid.New()
		}

		// update the deviceToken
		return bucket.Put([]byte(deviceInfo.DeviceKey), []byte(deviceInfo.DeviceToken))
	})

	if err != nil {
		logger.Errorf("route [/register] error: device registration failed: %v", err)
		return c.Status(500).JSON(failed(500, "device registration failed: %v", err))
	}

	logger.Infof("route [/register]: new device registered successfully, deviceKey %s, deviceToken %s", deviceInfo.DeviceKey, deviceInfo.DeviceToken)
	return c.Status(200).JSON(data(map[string]string{
		// compatible with old resp
		"key":          deviceInfo.DeviceKey,
		"device_key":   deviceInfo.DeviceKey,
		"device_token": deviceInfo.DeviceToken,
	}))
}
