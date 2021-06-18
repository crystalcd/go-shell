package main

import (
	rice "github.com/GeertJohan/go.rice"
	"github.com/gin-gonic/gin"
	"go-shell/webssh"
)

func main() {
	r := gin.Default()
	r.GET("/webssh", webssh.Webssh)
	r.StaticFS("/static", rice.MustFindBox("static").HTTPBox())
	//r.GET("/", func(c *gin.Context) {
	//	index, _ := rice.MustFindBox("static").Bytes("index.html")
	//	c.Data(http.StatusOK, "text/html", index)
	//})
	//r.GET("/webssh",webssh.Webssh)
	r.Run(":8081")
}
