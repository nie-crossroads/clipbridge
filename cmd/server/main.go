package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nie-crossroads/clipbridge/internal/app"
	"github.com/nie-crossroads/clipbridge/internal/config"
	"github.com/nie-crossroads/clipbridge/internal/repository/mysql"
	"github.com/nie-crossroads/clipbridge/internal/security"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("ClipBridge", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	isServe := len(args) == 0 || len(args) == 1 && args[0] == "serve"
	isAdmin := len(args) == 3 && args[0] == "admin" && args[1] == "create"
	if !isServe && !isAdmin {
		return errors.New("用法：clipbridge serve | clipbridge admin create <用户名>")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	store, err := mysql.Open(cfg.DB)
	if err != nil {
		return errors.New("数据库配置无效")
	}
	defer store.DB.Close()
	if isAdmin {
		return createAdmin(store, args[2])
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return app.Serve(ctx, cfg, store)
}

func createAdmin(store *mysql.Store, username string) error {
	if !app.ValidUsername(username) {
		return errors.New("用户名须为 1–64 个字符，首尾不能有空格")
	}
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return errors.New("请在交互式终端中创建管理员")
	}
	fmt.Fprint(os.Stderr, "密码（至少 12 字节）：")
	password, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}
	defer clear(password)
	if len(password) < 12 || len(password) > 72 {
		return errors.New("密码长度须为 12–72 字节")
	}
	fmt.Fprint(os.Stderr, "再次输入密码：")
	confirm, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}
	defer clear(confirm)
	if !bytes.Equal(password, confirm) {
		return errors.New("两次输入的密码不一致")
	}
	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.BootstrapAdmin(ctx, username, string(hash), security.RequestID()); err != nil {
		return errors.New("创建管理员失败，请检查数据库、建表状态及用户名是否已存在")
	}
	fmt.Fprintln(os.Stdout, "管理员已创建")
	return nil
}
