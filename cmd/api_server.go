package main

import (
	//_ "sensor-edge/docs"

	"github.com/gofiber/fiber/v2"
	//"github.com/gofiber/swagger"
)

func WebApp() {
	app := fiber.New()

	// 注册 swagger 路由
	//app.Get("/swagger/*", swagger.Handler)

	// 协议管理
	app.Get("/api/protocols", listProtocols)
	app.Post("/api/protocols", addProtocol)
	app.Put("/api/protocols/:id", updateProtocol)
	app.Delete("/api/protocols/:id", deleteProtocol)

	// 设备管理
	app.Get("/api/devices", listDevices)
	app.Post("/api/devices", addDevice)
	app.Put("/api/devices/:id", updateDevice)
	app.Delete("/api/devices/:id", deleteDevice)

	// 点位映射
	app.Get("/api/points/:device_id", listPoints)
	app.Post("/api/points/:device_id", importPoints)

	// 边缘规则
	app.Get("/api/rules", listRules)
	app.Post("/api/rules", addRule)
	app.Put("/api/rules/:id", updateRule)
	app.Delete("/api/rules/:id", deleteRule)

	// 上行通道
	app.Get("/api/uplinks", listUplinks)
	app.Post("/api/uplinks", addUplink)
	app.Put("/api/uplinks/:id", updateUplink)
	app.Delete("/api/uplinks/:id", deleteUplink)

	// 上行格式模板
	app.Get("/api/uplink_format", getUplinkFormat)
	app.Post("/api/uplink_format", setUplinkFormat)

	// 日志/调试
	app.Get("/api/logs", streamLogs)

	app.Listen(":8080")
}

// 以下为各API的空实现骨架
// listDevices godoc
// @Summary 获取设备列表
// @Tags Device
// @Produce  json
// @Success 200 {array} models.Device
// @Router /api/devices [get]
func listDevices(c *fiber.Ctx) error { return c.SendString("listDevices") }

// addDevice godoc
// @Summary 添加设备
// @Tags Device
// @Accept  json
// @Produce  json
// @Param device body models.Device true "设备信息"
// @Success 200 {object} models.Device
// @Router /api/devices [post]
func addDevice(c *fiber.Ctx) error { return c.SendString("addDevice") }

// updateDevice godoc
// @Summary 更新设备
// @Tags Device
// @Accept  json
// @Produce  json
// @Param id path string true "设备ID"
// @Param device body models.Device true "设备信息"
// @Success 200 {object} models.Device
// @Router /api/devices/{id} [put]
func updateDevice(c *fiber.Ctx) error { return c.SendString("updateDevice") }

// deleteDevice godoc
// @Summary 删除设备
// @Tags Device
// @Param id path string true "设备ID"
// @Success 200 {string} string "ok"
// @Router /api/devices/{id} [delete]
func deleteDevice(c *fiber.Ctx) error    { return c.SendString("deleteDevice") }
func listProtocols(c *fiber.Ctx) error   { return c.SendString("listProtocols") }
func addProtocol(c *fiber.Ctx) error     { return c.SendString("addProtocol") }
func updateProtocol(c *fiber.Ctx) error  { return c.SendString("updateProtocol") }
func deleteProtocol(c *fiber.Ctx) error  { return c.SendString("deleteProtocol") }
func listPoints(c *fiber.Ctx) error      { return c.SendString("listPoints") }
func importPoints(c *fiber.Ctx) error    { return c.SendString("importPoints") }
func listRules(c *fiber.Ctx) error       { return c.SendString("listRules") }
func addRule(c *fiber.Ctx) error         { return c.SendString("addRule") }
func updateRule(c *fiber.Ctx) error      { return c.SendString("updateRule") }
func deleteRule(c *fiber.Ctx) error      { return c.SendString("deleteRule") }
func listUplinks(c *fiber.Ctx) error     { return c.SendString("listUplinks") }
func addUplink(c *fiber.Ctx) error       { return c.SendString("addUplink") }
func updateUplink(c *fiber.Ctx) error    { return c.SendString("updateUplink") }
func deleteUplink(c *fiber.Ctx) error    { return c.SendString("deleteUplink") }
func getUplinkFormat(c *fiber.Ctx) error { return c.SendString("getUplinkFormat") }
func setUplinkFormat(c *fiber.Ctx) error { return c.SendString("setUplinkFormat") }
func streamLogs(c *fiber.Ctx) error      { return c.SendString("streamLogs") }
