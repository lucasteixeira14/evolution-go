package config

import (
	"database/sql"
	"os"
	"strconv"
	"time"

	"github.com/gomessguii/logger"
)

// Defaults iguais aos valores que estavam hardcoded em CreateUsersDB,
// CreateAuthDB e initPostgresAuthDB: sem as variaveis de ambiente o
// comportamento e identico ao do upstream.
const (
	defaultMaxOpenConns = 25
	defaultMaxIdleConns = 5
	defaultConnLifetime = 5 * time.Minute
	defaultConnIdleTime = 1 * time.Minute
)

// ApplyPoolSettings configura o pool de um *sql.DB a partir do ambiente.
//
// Existe porque o teto de conexoes precisa caber no limite do servidor, nao no
// que o app acha razoavel. Num Postgres compartilhado a role do evolution-go
// costuma ter CONNECTION LIMIT proprio, e cada processo abre mais de um pool
// (users via GORM, auth via sql.DB, e o sqlstore do whatsmeow que passou a
// reaproveitar o pool de auth). Com 25 por pool hardcoded, dois pools ja pedem
// 50 conexoes: quem tem limite menor que isso quebra sem aviso, e o sintoma
// aparece longe da causa — "no QR code available" em vez de erro de banco.
//
// PG_MAX_OPEN_CONNS=0 (ou negativo) mantem o comportamento do database/sql:
// sem teto. Valores invalidos caem no default com aviso.
func ApplyPoolSettings(db *sql.DB, name string) {
	if db == nil {
		return
	}

	maxOpen := envInt("PG_MAX_OPEN_CONNS", defaultMaxOpenConns)
	maxIdle := envInt("PG_MAX_IDLE_CONNS", defaultMaxIdleConns)

	// MaxIdleConns maior que MaxOpenConns e silenciosamente reduzido pelo
	// database/sql; deixar explicito evita config que mente sobre si mesma.
	if maxOpen > 0 && maxIdle > maxOpen {
		logger.LogWarn("[POOL] %s: PG_MAX_IDLE_CONNS=%d maior que PG_MAX_OPEN_CONNS=%d, usando %d", name, maxIdle, maxOpen, maxOpen)
		maxIdle = maxOpen
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(defaultConnLifetime)
	db.SetConnMaxIdleTime(defaultConnIdleTime)

	logger.LogInfo("[POOL] %s: max_open=%d max_idle=%d lifetime=%s idle_time=%s", name, maxOpen, maxIdle, defaultConnLifetime, defaultConnIdleTime)
}

func envInt(key string, def int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		logger.LogWarn("[POOL] %s=%q nao e um inteiro, usando %d", key, raw, def)
		return def
	}
	return v
}
