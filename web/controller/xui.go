package controller

import (
	"encoding/json"
	"net/http"

	"x-ui/logger"
	"x-ui/web/service"
	"x-ui/web/session"

	"github.com/gin-gonic/gin"
)

type XUIController struct {
	BaseController

	inboundController     *InboundController
	serverController      *ServerController
	settingController     *SettingController
	xraySettingController *XraySettingController
	serverService  service.ServerService
}

func NewXUIController(g *gin.RouterGroup) *XUIController {
	a := &XUIController{}
	a.initRouter(g)
	return a
}

func (a *XUIController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/panel")
	g.Use(a.checkLogin)

	g.GET("/", a.index)
	g.GET("/inbounds", a.inbounds)
	g.GET("/settings", a.settings)
	g.GET("/xray", a.xraySettings)
	g.GET("/navigation", a.navigation)

	g.GET("/userinfo", a.userinfoPage)
	g.GET("/api/user/info", a.userInfoAPI)

	g.GET("/servers", a.serversPage)

	a.inboundController = NewInboundController(g)
	a.serverController = NewServerController(g, a.serverService)
	a.settingController = NewSettingController(g)
	a.xraySettingController = NewXraySettingController(g)
}

func (a *XUIController) index(c *gin.Context) {
	html(c, "index.html", "pages.index.title", nil)
}

func (a *XUIController) inbounds(c *gin.Context) {
	html(c, "inbounds.html", "pages.inbounds.title", nil)
}

func (a *XUIController) settings(c *gin.Context) {
	html(c, "settings.html", "pages.settings.title", nil)
}

func (a *XUIController) xraySettings(c *gin.Context) {
	html(c, "xray.html", "pages.xray.title", nil)
}

func (a *XUIController) navigation(c *gin.Context) {
	html(c, "navigation.html", "pages.navigation.title", nil)
}

// 【新增 4】添加页面渲染方法
func (a *XUIController) serversPage(c *gin.Context) {
	html(c, "servers.html", "pages.controlledmanagement.title", nil)
}

func (a *XUIController) userinfoPage(c *gin.Context) {
	html(c, "userinfo.html", "pages.userinfo.title", nil)
}

func (a *XUIController) userInfoAPI(c *gin.Context) {
	user := session.GetLoginUser(c)
	if user == nil {
		pureJsonMsg(c, http.StatusUnauthorized, false, "unauthorized")
		return
	}

	inboundService := service.InboundService{}
	inbounds, err := inboundService.GetAllInbounds()
	if err != nil {
		jsonObj(c, nil, err)
		return
	}

	type UserInboundInfo struct {
		Remark     string `json:"remark"`
		Protocol   string `json:"protocol"`
		Up         int64  `json:"up"`
		Down       int64  `json:"down"`
		Total      int64  `json:"total"`
		ExpiryTime int64  `json:"expiryTime"`
		Enable     bool   `json:"enable"`
	}

	var userInbounds []UserInboundInfo

	traffic, err := inboundService.GetClientTrafficByEmail(user.Username)
	if err != nil {
		logger.Warningf("failed to get traffic for user %s: %v", user.Username, err)
	}

	for _, inbound := range inbounds {
		var settings map[string]any
		err := json.Unmarshal([]byte(inbound.Settings), &settings)
		if err != nil {
			continue
		}

		clientsInterface, ok := settings["clients"]
		if !ok {
			continue
		}

		clientsSlice, ok := clientsInterface.([]any)
		if !ok {
			continue
		}

		for _, ci := range clientsSlice {
			clientMap, ok := ci.(map[string]any)
			if !ok {
				continue
			}
			clientEmail, _ := clientMap["email"].(string)
			if clientEmail == user.Username {
				info := UserInboundInfo{
					Remark:   inbound.Remark,
					Protocol: string(inbound.Protocol),
				}
				if traffic != nil {
					info.Up = traffic.Up
					info.Down = traffic.Down
					info.Total = traffic.Total
					info.ExpiryTime = traffic.ExpiryTime
					info.Enable = traffic.Enable
				}
				userInbounds = append(userInbounds, info)
				break
			}
		}
	}

	jsonObj(c, userInbounds, nil)
}
