package rotateproxy

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

type ProxyURL struct {
	gorm.Model
	URL          string `gorm:"uniqueIndex;column:url"`
	Retry        int    `gorm:"column:retry"`
	Available    bool   `gorm:"column:available"`
	CanBypassGFW bool   `gorm:"column:can_bypass_gfw"`
}

func (ProxyURL) TableName() string {
	return "proxy_urls"
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	var err error
	DB, err = gorm.Open(sqlite.Open("db.db"), &gorm.Config{
		Logger: logger.Discard,
	})
	checkErr(err)
	DB.AutoMigrate(&ProxyURL{})
}

func CreateProxyURL(url string) error {
	tx := DB.Create(&ProxyURL{
		URL:       url,
		Retry:     0,
		Available: false,
	})
	return tx.Error
}

func QueryAvailProxyURL() (proxyURLs []ProxyURL, err error) {
	tx := DB.Where("available = ?", true).Find(&proxyURLs)
	err = tx.Error
	return
}

func QueryProxyURL() (proxyURLs []ProxyURL, err error) {
	tx := DB.Find(&proxyURLs)
	err = tx.Error
	return
}

func SetProxyURLAvail(url string, canBypassGFW bool) error {
	tx := DB.Model(&ProxyURL{}).Where("url = ?", url).Updates(ProxyURL{Retry: 0, Available: true, CanBypassGFW: canBypassGFW})
	return tx.Error
}

func SetProxyURLUnavail(url string) error {
	tx := DB.Model(&ProxyURL{}).Where("url = ?", url).Updates(ProxyURL{Retry: 0, Available: false})
	return tx.Error
}

func AddProxyURLRetry(url string) error {
	tx := DB.Model(&ProxyURL{}).Where("url = ?", url).Update("retry", gorm.Expr("retry + 1"))
	return tx.Error
}

func RandomProxyURL(regionFlag int) (pu string, markUnavail func(), err error) {
	var proxyURL ProxyURL
	var tx *gorm.DB
	switch regionFlag {
	case 1:
		tx = DB.Raw(fmt.Sprintf("SELECT * FROM %s WHERE available = ? AND can_bypass_gfw = ? ORDER BY RANDOM() LIMIT 1;", proxyURL.TableName()), true, false).Scan(&proxyURL)
	case 2:
		tx = DB.Raw(fmt.Sprintf("SELECT * FROM %s WHERE available = ? AND can_bypass_gfw = ? ORDER BY RANDOM() LIMIT 1;", proxyURL.TableName()), true, true).Scan(&proxyURL)
	default:
		tx = DB.Raw(fmt.Sprintf("SELECT * FROM %s WHERE available = 1 ORDER BY RANDOM() LIMIT 1;", proxyURL.TableName())).Scan(&proxyURL)
	}
	markUnavail = func() {
		SetProxyURLUnavail(proxyURL.URL)
	}
	return proxyURL.URL, markUnavail, tx.Error
}
