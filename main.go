package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/mysql"
	"github.com/congruity7/moonshot-go/pkg/api"
	"github.com/congruity7/moonshot-go/pkg/models"
	"github.com/congruity7/moonshot-go/pkg/service"
	"github.com/go-redis/redis"
	"github.com/joho/godotenv"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// type options struct{}

// type Option func(*options)

// type authMiddleWare struct {
// 	opts *options
// }

// var _ negroni.Handler = (*authMiddleWare)(nil)

// func NewAuthMiddleware(opts ...Option) negroni.Handler {
// 	aOpts := &options{}
// 	for _, opt := range opts {
// 		opt(aOpts)
// 	}
// 	return &authMiddleWare{opts: aOpts}
// }

// func (m *authMiddleWare) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
// 	return
// }

func goDotEnvVariable(key string) string {

	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		logrus.Error("Error loading .env file", err)
	}

	return os.Getenv(key)
}

func Setup() {
	dsn := fmt.Sprintf("%s:%s@%s(%s)/%s?parseTime=true",
		goDotEnvVariable("MOONSHOT_DB_USER"),
		goDotEnvVariable("MOONSHOT_DB_PASSWD"),
		goDotEnvVariable("MOONSHOT_DB_PROTOCOL"),
		goDotEnvVariable("MOONSHOT_DB_INSTANCE"),
		goDotEnvVariable("MOONSHOT_DB_NAME"))

	logrus.Info("DSN : ", dsn)

	//redisTlsCert := tls.LoadX509KeyPair(("REDIS_TLS_CERT"))

	dbInstance, err := gorm.Open(mysql.New(mysql.Config{
		DriverName: "mysql",
		DSN:        dsn,
	}), &gorm.Config{})

	if err != nil {
		logrus.Fatal("connecting to database : ", err)
	}

	logrus.Info("success connecting to db")

	dbInstance.AutoMigrate(&models.User{}, &models.Wallet{}, &models.Round{}, &models.PlacedBet{}, &models.Config{})

	redisHost := goDotEnvVariable("MOONSHOT_REDIS_HOST")
	if redisHost == "" {
		logrus.Fatal("connecting to redis : ", errors.New("redis host needs to be set"))
	}
	redisPort := goDotEnvVariable("MOONSHOT_REDIS_PORT")
	if redisPort == "" {
		logrus.Fatal("connecting to redis : ", errors.New("redis port needs to be set"))
	}
	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)

	certs := x509.NewCertPool()
	ok := certs.AppendCertsFromPEM([]byte(goDotEnvVariable("REDIS_TLS_CERT")))

	if !ok {
		logrus.Fatal("adding certificate")
	}

	tlsConfig := tls.Config{
		RootCAs: certs,
	}

	// const maxConnections = 10
	// redisPool := &redis.Pool{
	// 	MaxIdle: maxConnections,
	// 	Dial: func() (redis.Conn, error) {
	// 		c, err := redis.Dial("tcp", redisAddr)
	// 		if err != nil {
	// 			return nil, fmt.Errorf("redis.Dial: %v", err)
	// 		}
	// 		return c, err
	// 	},
	// }
	// ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	// defer cancel()

	// conn := redis.NewClient()
	// defer conn.Close()

	//i, err := redis.String(conn.Do("PING"))
	// if conn == nil {
	// 	logrus.Error("getting connection")
	// }

	// i, err := redis.String(conn.Do("PING"))

	// if err != nil {
	// 	logrus.Error("pinging the memory store.", err)
	// }

	// logrus.Info("PING ", i)
	rdb := redis.NewClient(&redis.Options{
		Addr:       redisAddr,
		Password:   "", // no password set
		DB:         0,  // use default DB
		MaxRetries: 10,
		TLSConfig:  &tlsConfig,
	})
	defer rdb.Close()

	var wg sync.WaitGroup
	var stopChan <-chan struct{}

	ds := service.NewDatabaseService(dbInstance)
	rs := service.NewRedisService(rdb)
	logger := logrus.New()

	wg.Add(2)

	go StartAPI(&wg, ds, rs, logger, stopChan)
	//go SyncTransactionsFromRPC(&wg, ds, rs, logger, stopChan)

	wg.Wait()
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func StartAPI(wg *sync.WaitGroup, ds *service.DatabaseService, rs *service.RedisService, logger *logrus.Logger, stopChan <-chan struct{}) {
	defer wg.Done()

	router := httprouter.New()

	ac := api.NewAPIContext(ds, rs, logger)
	router.GET("/", Index)
	router.GET("/moonshot/v1/user/:user_id/", ac.GetUserByID)
	router.GET("/moonshot/v1/user", ac.GetUsers)
	router.POST("/moonshot/v1/user", ac.CreateUser)
	router.PUT("/moonshot/v1/user", ac.UpdateUser)
	router.DELETE("/moonshot/v1/user/:user_id/", ac.DeleteUserByID)

	router.GET("/moonshot/v1/wallet/:wallet_id/", ac.GetWalletByID)
	router.GET("/moonshot/v1/wallet", ac.GetWallets)
	router.POST("/moonshot/v1/wallet", ac.CreateWallet)
	router.PUT("/moonshot/v1/wallet", ac.UpdateWallet)
	router.DELETE("/moonshot/v1/wallet/:wallet/", ac.DeleteWalletByID)

	router.GET("/moonshot/v1/config", ac.GetConfig)
	router.POST("/moonshot/v1/config", ac.CreateConfig)
	router.PUT("/moonshot/v1/config", ac.UpdateConfig)

	//router.GET("/moonshot/v1/keys/test", ac.PingStore)
	router.GET("/moonshot/v1/keys/:id", ac.GetKey)
	router.POST("/moonshot/v1/keys/:id", ac.CreateKey)
	router.PUT("/moonshot/v1/keys/:id", ac.CreateKey)
	router.DELETE("/moonshot/v1/keys/:id", ac.DeleteKey)

	router.GET("/moonshot/v1/history", ac.GetBetHistory)
	router.POST("/moonshot/v1/history", ac.CreateBetHistory)
	// n := negroni.New(negroni.NewRecovery(),
	// 	NewAuthMiddleware())
	// n.UseHandler(router)

	//api := &http.Server{Addr: ":8000", Handler: n}

	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8000"
		}
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), router))
		// if err := http.ListenAndServe(":8000", router); err != nil {
		// 	logger.Error("starting api server", err)
		// }
	}()

	logrus.Info("waiting for stop signal")

	<-stopChan

	logger.Info("shutting down api server")

	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	// defer cancel()
	// api.Shutdown(ctx)

}

// func SyncTransactionsFromRPC(wg *sync.WaitGroup, ds *service.DatabaseService, rs *service.RedisService, logger *logrus.Logger, stopChan <-chan struct{}) {
// 	defer wg.Done()
// 	periodTicker := time.NewTicker(45 * time.Second)

// 	for {
// 		select {
// 		case <-periodTicker.C:
// 			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// 			defer cancel()

// 			u := url.URL{Scheme: "ws", Host: "api.devnet.solana.com"}
// 			log.Printf("connecting to %s", u.String())

// 			c, rsp, err := websocket.DefaultDialer.DialContext(ctx, u.String(), http.Header{"Content-Type": []string{"application/json"}})
// 			if err != nil {
// 				log.Fatal("dial:", err)
// 			}

// 			logrus.Info("connected", rsp.Body)

// 			objStream := websocketjsonrpc2.NewObjectStream(c)

// 			conn := jsonrpc2.NewConn(ctx, objStream, nil)

// 			var result interface{}
// 			rpcVersion := map[string]interface{}{"jsonrpc": "2.0"}
// 			rpcID := map[string]interface{}{"id": 1}

// 			err = conn.Call(ctx, "getBalance", []interface{}{"83astBRguLMdt2h5U1Tpdq5tjFoJ6noeGwaY3mDLVcri", rpcVersion, rpcID}, result)

// 			if err != nil {
// 				logger.Error(err)
// 			}

// 			logrus.Info("result", result)

// 			conn.Close()

// 		case <-stopChan:

// 		}
// 	}

// }

func main() {
	Setup()
}
