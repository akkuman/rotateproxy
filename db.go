package rotateproxy

import (
	"fmt"
	"os"

	"github.com/glebarez/sqlite"
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
	Timeout      int64  `gorm:"column:timeout;default:0"`
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
	// 启动前删除缓存数据库
	if err = os.Remove("db.db"); err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
	}
	DB, err = gorm.Open(sqlite.Open("db.db"), &gorm.Config{
		Logger: logger.Discard,
	})
	checkErr(err)

	// According to https://github.com/mattn/go-sqlite3/issues/274#issuecomment-191597862,
	// solve "database is locked" error using `db.SetMaxOpenConns(1)`
	sqlDB, err := DB.DB()
	checkErr(err)
	sqlDB.SetMaxOpenConns(1)

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

func SetProxyURLAvail(url string, timeout int64, canBypassGFW bool) error {
	tx := DB.Model(&ProxyURL{}).Where("url = ?", url).Updates(ProxyURL{
		Retry:        0,
		Available:    true,
		CanBypassGFW: canBypassGFW,
		Timeout:      timeout,
	})
	return tx.Error
}

func SetProxyURLUnavail(url string) error {
	tx := DB.Model(&ProxyURL{}).Where("url = ?", url).Updates(ProxyURL{Retry: 0, Available: false})
	ErrorLog(Notice("Mark %v Unavailble! ", url))
	return tx.Error
}

func AddProxyURLRetry(url string) error {
	tx := DB.Model(&ProxyURL{}).Where("url = ?", url).Update("retry", gorm.Expr("retry + 1"))
	return tx.Error
}

func RandomProxyURL(regionFlag int, strategyFlag int) (pu string, markUnavail func(), err error) {
	var proxyURL ProxyURL
	var tx *gorm.DB
	if strategyFlag == 0 {
		switch regionFlag {
		case 1:
			tx = DB.Raw(fmt.Sprintf("SELECT * FROM %s WHERE available = ? AND can_bypass_gfw = ? ORDER BY RANDOM() LIMIT 1;", proxyURL.TableName()), true, false).Scan(&proxyURL)

		case 2:
			tx = DB.Raw(fmt.Sprintf("SELECT * FROM %s WHERE available = ? AND can_bypass_gfw = ? ORDER BY RANDOM() LIMIT 1;", proxyURL.TableName()), true, true).Scan(&proxyURL)
		default:
			tx = DB.Raw(fmt.Sprintf("SELECT * FROM %s WHERE available = 1 ORDER BY RANDOM() LIMIT 1;", proxyURL.TableName())).Scan(&proxyURL)
		}
	} else {
		switch regionFlag {
		case 1:
			tx = DB.Raw(fmt.Sprintf("SELECT * FROM (SELECT * FROM %s WHERE available = 1 AND can_bypass_gfw = ? AND timeout <> 0 ORDER BY timeout LIMIT ?) TA ORDER BY RANDOM() LIMIT 1;", proxyURL.TableName()), false, strategyFlag).Scan(&proxyURL)
		case 2:
			tx = DB.Raw(fmt.Sprintf("SELECT * FROM (SELECT * FROM %s WHERE available = 1 AND can_bypass_gfw = ? AND timeout <> 0 ORDER BY timeout LIMIT ?) TA ORDER BY RANDOM() LIMIT 1;", proxyURL.TableName()), true, strategyFlag).Scan(&proxyURL)
		default:
			tx = DB.Raw(fmt.Sprintf("SELECT * FROM (SELECT * FROM %s WHERE available = 1 AND timeout <> 0 ORDER BY timeout LIMIT ?) TA ORDER BY RANDOM() LIMIT 1;", proxyURL.TableName()), strategyFlag).Scan(&proxyURL)
		}
	}
	pu = proxyURL.URL
	err = tx.Error
	markUnavail = func() {
		SetProxyURLUnavail(proxyURL.URL)
	}
	return
}
