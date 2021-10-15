package rotateproxy

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

type ProxyURL struct {
	gorm.Model
	URL       string `gorm:"uniqueIndex;column:url"`
	Retry     int    `gorm:"column:retry"`
	Available bool   `gorm:"column:available"`
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	var err error
	DB, err = gorm.Open(sqlite.Open("db.db"), &gorm.Config{})
	checkErr(err)
	DB.AutoMigrate(&ProxyURL{})
}

func CreateProxyURL(url string) {
	DB.Create(&ProxyURL{
		URL:       url,
		Retry:     0,
		Available: false,
	})
}

func QueryAvailProxyURL() (proxyURLs []ProxyURL, err error) {
	tx := DB.Where("available = ?", true).Find(&proxyURLs)
	err = tx.Error
	return
}

func SetProxyURLAvail(url string) error {
	tx := DB.Model(&ProxyURL{}).Where("url = ?", url).Updates(ProxyURL{Retry: 0, Available: true})
	return tx.Error
}
