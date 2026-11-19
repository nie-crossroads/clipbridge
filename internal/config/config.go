package config

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	driver "github.com/go-sql-driver/mysql"
)

type Config struct {
	Addr               string
	SecureCookie       bool
	DB                 *driver.Config
	LocalAdminUsername string
	LocalAdminPassword string
}

func Load() (Config, error) {
	local := map[string]string{}
	file, err := os.Open(".env")
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			key, value, ok := strings.Cut(strings.TrimSpace(scanner.Text()), "=")
			if !ok || strings.HasPrefix(key, "#") {
				continue
			}
			value = strings.TrimSpace(value)
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}
			local[strings.TrimSpace(key)] = value
		}
		if err := scanner.Err(); err != nil {
			return Config{}, fmt.Errorf("read .env: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("open .env: %w", err)
	}

	get := func(key, fallback string) string {
		key = "CLIPBRIDGE_" + key
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
		if value, ok := local[key]; ok {
			return value
		}
		return fallback
	}
	db := driver.NewConfig()
	db.Net = "tcp"
	db.Addr = net.JoinHostPort(get("DB_HOST", "127.0.0.1"), get("DB_PORT", "3306"))
	db.DBName = get("DB_NAME", "clipbridge")
	db.User = get("DB_USER", "root")
	db.Passwd = get("DB_PASSWORD", "")
	db.TLSConfig = get("DB_TLS", "false")
	db.ParseTime = true
	db.Params = map[string]string{"time_zone": "'+00:00'"}
	addr := get("ADDR", "127.0.0.1:8080")
	secure, err := strconv.ParseBool(get("SECURE_COOKIE", "true"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid CLIPBRIDGE_SECURE_COOKIE")
	}
	return Config{
		Addr: addr, SecureCookie: secure, DB: db,
		LocalAdminUsername: get("LOCAL_ADMIN_USERNAME", ""),
		LocalAdminPassword: get("LOCAL_ADMIN_PASSWORD", ""),
	}, nil
}
