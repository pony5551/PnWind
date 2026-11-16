package router

import (
	"github.com/pony5551/PnWind/core/pnlibs"

	"github.com/gofiber/fiber/v3"
)

type RouterItem struct {
	Name   string `json:"name"`
	Prefix string `json:"prefix"`
}

type Router interface {
	GetItem() *RouterItem
	Setup(app *fiber.App, pool *pnlibs.PnObjects[string, any])
}
