package healthcheck

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ---------- PostgresCheck tests ----------

func TestNewPostgresCheck(t *testing.T) {
	check := NewPostgresCheck(&gorm.DB{})
	assert.NotNil(t, check)
	assert.Equal(t, "postgres", check.Name())
}

func TestPostgresCheck_Name(t *testing.T) {
	check := &PostgresCheck{}
	assert.Equal(t, "postgres", check.Name())
}

func TestPostgresCheck_Pass_Success(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)
	defer db.Close()

	// gorm.Open might ping once during initialization, so we mock it.
	mock.ExpectPing().WillReturnError(nil)
	// check.Pass() will ping a second time.
	mock.ExpectPing().WillReturnError(nil)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	assert.NoError(t, err)

	check := NewPostgresCheck(gormDB)
	assert.True(t, check.Pass())
}

func TestPostgresCheck_Pass_Fail(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)
	defer db.Close()

	// gorm.Open succeeds
	mock.ExpectPing().WillReturnError(nil)
	// check.Pass() fails
	mock.ExpectPing().WillReturnError(errors.New("db down"))

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	assert.NoError(t, err)

	check := NewPostgresCheck(gormDB)
	assert.False(t, check.Pass())
}

// ---------- RedisCheck tests ----------

func TestNewRedisCheck(t *testing.T) {
	check := NewRedisCheck(&redis.Client{})
	assert.NotNil(t, check)
	assert.Equal(t, "redis", check.Name())
}

func TestRedisCheck_Name(t *testing.T) {
	check := &RedisCheck{}
	assert.Equal(t, "redis", check.Name())
}

func TestRedisCheck_Pass_Success(t *testing.T) {
	client, mock := redismock.NewClientMock()
	mock.ExpectPing().SetVal("PONG")

	check := NewRedisCheck(client)
	assert.True(t, check.Pass())
}

func TestRedisCheck_Pass_Fail(t *testing.T) {
	client, mock := redismock.NewClientMock()
	mock.ExpectPing().SetErr(errors.New("redis down"))

	check := NewRedisCheck(client)
	assert.False(t, check.Pass())
}

// ---------- Register tests ----------

func TestRegister_NilDeps_RegistersEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	Register(r, nil, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRegister_WithDeps_RegistersEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	db, mockDB, _ := sqlmock.New(sqlmock.MonitorPingsOption(true))
	defer db.Close()
	// Gorm init ping
	mockDB.ExpectPing().WillReturnError(nil)
	gormDB, _ := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})

	redisClient, mockRedis := redismock.NewClientMock()
	mockRedis.ExpectPing().SetVal("PONG")

	Register(r, gormDB, redisClient)

	// Since we execute the HTTP request, the checkers will run. We need to set up expectations for them.
	// We need expectations for Postgres and Redis Ping when the health check endpoint is called.
	mockDB.ExpectPing().WillReturnError(nil)
	mockRedis.ExpectPing().SetVal("PONG")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
