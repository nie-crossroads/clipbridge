package app

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/nie-crossroads/clipbridge/internal/config"
	"github.com/nie-crossroads/clipbridge/internal/repository/mysql"
	"github.com/nie-crossroads/clipbridge/internal/security"
	"golang.org/x/crypto/bcrypt"
)

func syncLocalAdmin(ctx context.Context, cfg config.Config, store *mysql.Store) error {
	if !loopbackAddress(cfg.Addr) || cfg.DB.Net != "tcp" || !loopbackAddress(cfg.DB.Addr) {
		return errors.New("启动密码同步要求服务与数据库均使用本机回环地址")
	}
	password := []byte(cfg.LocalAdminPassword)
	defer clear(password)
	if !ValidUsername(cfg.LocalAdminUsername) || len(password) < 6 || len(password) > 72 {
		return errors.New("启动密码同步需要配置管理员用户名和 6–72 字节的密码")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	admin, err := store.FindAdmin(ctx, cfg.LocalAdminUsername)
	if err != nil {
		return errors.New("启动密码同步无法读取管理员，请检查本地数据库及账号状态")
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), password) == nil {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return errors.New("生成管理员密码哈希失败")
	}
	if err := store.ResetAdminPassword(ctx, admin, string(hash), security.RequestID()); err != nil {
		return errors.New("启动密码同步失败，请检查本地数据库")
	}
	return nil
}

func loopbackAddress(address string) bool {
	host, _, err := net.SplitHostPort(address)
	return err == nil && (host == "localhost" || net.ParseIP(host).IsLoopback())
}
