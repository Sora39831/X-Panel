package model

import (
	"fmt"

	"x-ui/util/json_util"
	"x-ui/xray"
)

type Protocol string

const (
	VMESS       Protocol = "vmess"
	VLESS       Protocol = "vless"
	Tunnel      Protocol = "tunnel"
	HTTP        Protocol = "http"
	Trojan      Protocol = "trojan"
	Shadowsocks Protocol = "shadowsocks"
	Socks       Protocol = "socks"
	WireGuard   Protocol = "wireguard"
)

type User struct {
	Id       int    `json:"id" gorm:"primaryKey;autoIncrement" bson:"_id,omitempty"`
	Username string `json:"username" bson:"username"`
	Password string `json:"password" bson:"password"`
}

type Inbound struct {
	Id          int                  `json:"id" form:"id" gorm:"primaryKey" bson:"_id,omitempty"`
	UserId      int                  `json:"-" bson:"user_id"`
	Up          int64                `json:"up" form:"up" bson:"up"`
	Down        int64                `json:"down" form:"down" bson:"down"`
	Total       int64                `json:"total" form:"total" bson:"total"`
	AllTime     int64                `json:"allTime" form:"allTime" gorm:"default:0" bson:"all_time"`
	Remark      string               `json:"remark" form:"remark" bson:"remark"`
	Enable      bool                 `json:"enable" form:"enable" bson:"enable"`
	ExpiryTime  int64                `json:"expiryTime" form:"expiryTime" bson:"expiry_time"`

	// 中文注释: 新增设备限制字段，用于存储每个入站的设备数限制。
	// gorm:"column:device_limit;default:0" 定义了数据库中的字段名和默认值。
	DeviceLimit   int                  `json:"deviceLimit" form:"deviceLimit" gorm:"column:device_limit;default:0" bson:"device_limit"`

	ClientStats []xray.ClientTraffic `gorm:"foreignKey:InboundId;references:Id" json:"clientStats" form:"clientStats" bson:"-"`

	// config part
	Listen         string   `json:"listen" form:"listen" bson:"listen"`
	Port           int      `json:"port" form:"port" bson:"port"`
	Protocol       Protocol `json:"protocol" form:"protocol" bson:"protocol"`
	Settings       string   `json:"settings" form:"settings" bson:"settings"`
	StreamSettings string   `json:"streamSettings" form:"streamSettings" bson:"stream_settings"`
	Tag            string   `json:"tag" form:"tag" gorm:"unique" bson:"tag"`
	Sniffing       string   `json:"sniffing" form:"sniffing" bson:"sniffing"`
}

type OutboundTraffics struct {
	Id    int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement" bson:"_id,omitempty"`
	Tag   string `json:"tag" form:"tag" gorm:"unique" bson:"tag"`
	Up    int64  `json:"up" form:"up" gorm:"default:0" bson:"up"`
	Down  int64  `json:"down" form:"down" gorm:"default:0" bson:"down"`
	Total int64  `json:"total" form:"total" gorm:"default:0" bson:"total"`
}

type InboundClientIps struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement" bson:"_id,omitempty"`
	ClientEmail string `json:"clientEmail" form:"clientEmail" gorm:"unique" bson:"client_email"`
	Ips         string `json:"ips" form:"ips" bson:"ips"`
}

type HistoryOfSeeders struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement" bson:"_id,omitempty"`
	SeederName string `json:"seederName" bson:"seeder_name"`
}

func (i *Inbound) GenXrayInboundConfig() *xray.InboundConfig {
	listen := i.Listen
	if listen != "" {
		listen = fmt.Sprintf("\"%v\"", listen)
	}
	return &xray.InboundConfig{
		Listen:         json_util.RawMessage(listen),
		Port:           i.Port,
		Protocol:       string(i.Protocol),
		Settings:       json_util.RawMessage(i.Settings),
		StreamSettings: json_util.RawMessage(i.StreamSettings),
		Tag:            i.Tag,
		Sniffing:       json_util.RawMessage(i.Sniffing),
	}
}

type Setting struct {
	Id    int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement" bson:"_id,omitempty"`
	Key   string `json:"key" form:"key" bson:"key"`
	Value string `json:"value" form:"value" bson:"value"`
}

type Client struct {
	ID         string `json:"id"`
	Security   string `json:"security"`
	Password   string `json:"password"`
	
	// 中文注释: 新增“限速”字段，单位 KB/s，0 表示不限速。
    SpeedLimit   int           `json:"speedLimit" form:"speedLimit"`
	
	Flow       string `json:"flow"`
	Email      string `json:"email"`
	LimitIP    int    `json:"limitIp"`
	TotalGB    int64  `json:"totalGB" form:"totalGB"`
	ExpiryTime int64  `json:"expiryTime" form:"expiryTime"`
	Enable     bool   `json:"enable" form:"enable"`
	TgID       int64  `json:"tgId" form:"tgId"`
	SubID      string `json:"subId" form:"subId"`
	Comment    string `json:"comment" form:"comment"`
	Reset      int    `json:"reset" form:"reset"`
	CreatedAt  int64  `json:"created_at,omitempty"`
	UpdatedAt  int64  `json:"updated_at,omitempty"`
}

type VLESSSettings struct {
	Clients    []Client `json:"clients"`
	Decryption string   `json:"decryption"`
	Encryption string   `json:"encryption"`
	Fallbacks  []any    `json:"fallbacks"`
}
