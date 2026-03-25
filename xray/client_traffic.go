package xray

type ClientTraffic struct {
	Id         int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement" bson:"_id,omitempty"`
	InboundId  int    `json:"inboundId" form:"inboundId" bson:"inbound_id"`
	Enable     bool   `json:"enable" form:"enable" bson:"enable"`
	Email      string `json:"email" form:"email" gorm:"unique" bson:"email"`
	Up         int64  `json:"up" form:"up" bson:"up"`
	Down       int64  `json:"down" form:"down" bson:"down"`
	AllTime    int64  `json:"allTime" form:"allTime" bson:"all_time"`
	ExpiryTime int64  `json:"expiryTime" form:"expiryTime" bson:"expiry_time"`
	Total      int64  `json:"total" form:"total" bson:"total"`
	Reset      int    `json:"reset" form:"reset" gorm:"default:0" bson:"reset"`
	LastOnline int64  `json:"lastOnline" form:"lastOnline" gorm:"default:0" bson:"last_online"`
}
